package respond

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// Responder writes a complete response. Errors are only logged, except notWrittenError.
type Responder interface {
	WriteResponse(w http.ResponseWriter, r *http.Request) error
}

// notWrittenError is a failure before writing, so it can still be answered.
type notWrittenError struct {
	err error
}

func (e *notWrittenError) Error() string {
	return e.err.Error()
}

func (e *notWrittenError) Unwrap() error {
	return e.err
}

// Handler adapts fn to an [http.HandlerFunc]. Expected failures are a Responder such as
// [ErrorResponse]; every error is unexpected and answered with a 500, logged and reported to
// Sentry, unless the client cancelled the request.
func Handler(fn func(r *http.Request) (Responder, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r)
		Write(w, r, resp, err)
	}
}

// Middleware adapts fn to a middleware. fn returns the request to serve the next handler with, or
// answers in its place like [Handler].
func Middleware(fn func(r *http.Request) (*http.Request, Responder, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if nextReq, resp, err := fn(r); nextReq != nil && resp == nil && err == nil {
				next.ServeHTTP(w, nextReq)
			} else {
				Write(w, r, resp, err)
			}
		})
	}
}

// Write answers like [Handler], for code that writes to w itself.
func Write(w http.ResponseWriter, r *http.Request, resp Responder, err error) {
	if err == nil && resp == nil {
		err = errors.New("handler returned neither a response nor an error")
	}
	if err != nil {
		internalError(w, r, err)
	} else if err := resp.WriteResponse(w, r); err != nil {
		if notWritten, ok := errors.AsType[*notWrittenError](err); ok {
			internalError(w, r, notWritten.err)
		} else {
			internalctx.GetLogger(r.Context()).Warn("failed to write response", zap.Error(err))
		}
	}
}

func internalError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		log.Debug("request canceled", zap.Error(err))
	} else {
		log.Error("request failed", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeHeader(w http.ResponseWriter, header http.Header, status int) {
	maps.Copy(w.Header(), header)
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
}

func withHeader(header http.Header, key, value string) http.Header {
	header = header.Clone()
	if header == nil {
		header = http.Header{}
	}
	header.Set(key, value)
	return header
}

// JSONResponse sends Body as JSON. Status defaults to 200.
type JSONResponse struct {
	Status int
	Header http.Header
	Body   any
}

func JSON(body any) JSONResponse {
	return JSONResponse{Body: body}
}

func (resp JSONResponse) WithStatus(status int) JSONResponse {
	resp.Status = status
	return resp
}

func (resp JSONResponse) WithHeader(key, value string) JSONResponse {
	resp.Header = withHeader(resp.Header, key, value)
	return resp
}

func (resp JSONResponse) WriteResponse(w http.ResponseWriter, r *http.Request) error {
	// Encode first, so a failure can still become a 500.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp.Body); err != nil {
		return &notWrittenError{err}
	}
	w.Header().Set("Content-Type", "application/json")
	writeHeader(w, resp.Header, resp.Status)
	_, err := buf.WriteTo(w)
	return err
}

// StreamResponse copies Body and closes it if possible. Defaults: 200, application/octet-stream.
type StreamResponse struct {
	Status      int
	ContentType string
	Header      http.Header
	Body        io.Reader
}

func Stream(contentType string, body io.Reader) StreamResponse {
	return StreamResponse{ContentType: contentType, Body: body}
}

func (resp StreamResponse) WithStatus(status int) StreamResponse {
	resp.Status = status
	return resp
}

func (resp StreamResponse) WithHeader(key, value string) StreamResponse {
	resp.Header = withHeader(resp.Header, key, value)
	return resp
}

func (resp StreamResponse) WriteResponse(w http.ResponseWriter, r *http.Request) error {
	if closer, ok := resp.Body.(io.Closer); ok {
		defer closer.Close()
	}
	writeStreamHeader(w, resp.ContentType, resp.Header, resp.Status)
	_, err := io.Copy(w, resp.Body)
	return err
}

func writeStreamHeader(w http.ResponseWriter, contentType string, header http.Header, status int) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	writeHeader(w, header, status)
}

// WriterResponse lets WriteBody write the body while producing it. Status and headers are sent with
// the first write, so until then WriteBody can answer in its place like a [Handler]. After it, an
// error is only logged and a Responder is ignored. Defaults: 200, application/octet-stream.
type WriterResponse struct {
	Status      int
	ContentType string
	Header      http.Header
	WriteBody   func(w io.Writer) (Responder, error)
}

func Writer(contentType string, writeBody func(w io.Writer) (Responder, error)) WriterResponse {
	return WriterResponse{ContentType: contentType, WriteBody: writeBody}
}

func (resp WriterResponse) WithStatus(status int) WriterResponse {
	resp.Status = status
	return resp
}

func (resp WriterResponse) WithHeader(key, value string) WriterResponse {
	resp.Header = withHeader(resp.Header, key, value)
	return resp
}

func (resp WriterResponse) WriteResponse(w http.ResponseWriter, r *http.Request) error {
	body := &deferredHeaderWriter{w: w, resp: resp}
	if answer, err := resp.WriteBody(body); body.started {
		return err
	} else if err != nil {
		return &notWrittenError{err}
	} else if answer != nil {
		return answer.WriteResponse(w, r)
	} else {
		body.start()
		return nil
	}
}

type deferredHeaderWriter struct {
	w       http.ResponseWriter
	resp    WriterResponse
	started bool
}

func (d *deferredHeaderWriter) Write(p []byte) (int, error) {
	if !d.started {
		d.start()
	}
	return d.w.Write(p)
}

func (d *deferredHeaderWriter) start() {
	d.started = true
	writeStreamHeader(d.w, d.resp.ContentType, d.resp.Header, d.resp.Status)
}

// ErrorResponse sends Message as plain text. The frontend displays it verbatim, so write it for
// the end user. It is deliberately no error: an error returned from a handler is always a 500.
type ErrorResponse struct {
	Status  int
	Message string
}

func Error(status int, message string) ErrorResponse {
	return ErrorResponse{Status: status, Message: message}
}

func BadRequest(message string) ErrorResponse {
	return Error(http.StatusBadRequest, message)
}

func Unauthorized(message string) ErrorResponse {
	return Error(http.StatusUnauthorized, message)
}

func Forbidden(message string) ErrorResponse {
	return Error(http.StatusForbidden, message)
}

func NotFound(message string) ErrorResponse {
	return Error(http.StatusNotFound, message)
}

func Conflict(message string) ErrorResponse {
	return Error(http.StatusConflict, message)
}

func (resp ErrorResponse) WriteResponse(w http.ResponseWriter, r *http.Request) error {
	http.Error(w, resp.Message, resp.Status)
	return nil
}

// StatusResponse sends a status without a body.
type StatusResponse struct {
	Status int
	Header http.Header
}

func Status(status int) StatusResponse {
	return StatusResponse{Status: status}
}

func NoContent() StatusResponse {
	return Status(http.StatusNoContent)
}

func (resp StatusResponse) WithHeader(key, value string) StatusResponse {
	resp.Header = withHeader(resp.Header, key, value)
	return resp
}

func (resp StatusResponse) WriteResponse(w http.ResponseWriter, r *http.Request) error {
	writeHeader(w, resp.Header, resp.Status)
	return nil
}

// RedirectResponse redirects to URL. Status defaults to 302.
type RedirectResponse struct {
	URL    string
	Status int
}

func Redirect(url string, status int) RedirectResponse {
	return RedirectResponse{URL: url, Status: status}
}

func (resp RedirectResponse) WriteResponse(w http.ResponseWriter, r *http.Request) error {
	status := resp.Status
	if status == 0 {
		status = http.StatusFound
	}
	http.Redirect(w, r, resp.URL, status)
	return nil
}
