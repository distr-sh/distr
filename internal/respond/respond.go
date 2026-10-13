package respond

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
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

// Handler adapts fn to an [http.HandlerFunc]. Errors are answered as:
//
//   - [*apierrors.Error]: its status and message
//   - wrapped sentinel: its status, generic text (may carry internals)
//   - client cancellation: nothing
//   - anything else: 500, logged and reported to Sentry
func Handler(fn func(r *http.Request) (Responder, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r)
		if err == nil && resp == nil {
			err = errors.New("handler returned neither a response nor an error")
		}
		if err != nil {
			Error(w, r, err)
			return
		}

		if err := resp.WriteResponse(w, r); err != nil {
			if notWritten, ok := errors.AsType[*notWrittenError](err); ok {
				Error(w, r, notWritten.err)
			} else {
				internalctx.GetLogger(r.Context()).Warn("failed to write response", zap.Error(err))
			}
		}
	}
}

// Error answers err like [Handler], for code that writes to w itself.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	if apiErr, ok := errors.AsType[*apierrors.Error](err); ok {
		http.Error(w, apiErr.Message, apiErr.Status)
	} else if status, ok := apierrors.SentinelStatus(err); ok {
		log.Debug("request rejected", zap.Int("status", status), zap.Error(err))
		http.Error(w, http.StatusText(status), status)
	} else if errors.Is(err, context.Canceled) && ctx.Err() != nil {
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
type JSONResponse[T any] struct {
	Status int
	Header http.Header
	Body   T
}

func JSON[T any](body T) JSONResponse[T] {
	return JSONResponse[T]{Body: body}
}

func (resp JSONResponse[T]) WithStatus(status int) JSONResponse[T] {
	resp.Status = status
	return resp
}

func (resp JSONResponse[T]) WithHeader(key, value string) JSONResponse[T] {
	resp.Header = withHeader(resp.Header, key, value)
	return resp
}

func (resp JSONResponse[T]) WriteResponse(w http.ResponseWriter, r *http.Request) error {
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
	contentType := resp.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	writeHeader(w, resp.Header, resp.Status)
	_, err := io.Copy(w, resp.Body)
	return err
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
