package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/contenttype"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/validation"
)

const maxFilterLength = 200

type TimeseriesRequest struct {
	TimeseriesRangeRequest
	Limit *int `query:"limit"`
}

// TimeseriesRangeRequest are the parameters that select a subset of a timeseries. Exports
// accept them too, but no limit.
type TimeseriesRangeRequest struct {
	Before *time.Time `query:"before"`
	After  *time.Time `query:"after"`
	Filter *string    `query:"filter"`
}

// timeseriesRange is the raw range selection of a timeseries read or export request.
type timeseriesRange struct {
	// Before is the inclusive end of the range. A zero value means "now".
	Before time.Time
	// After is the inclusive start of the range. A zero value means "not specified by the
	// client", which every caller defaults differently.
	After time.Time
	// Filter is an optional RE2 filter on the record body, already validated.
	Filter string
}

// defaultTimeseriesLimit is the page size used when a timeseries read does not request one.
const defaultTimeseriesLimit = 25

// parseTimeseriesLimit parses the limit query parameter of a timeseries read. All returned
// errors are caused by invalid client input and should be answered with status 400.
func parseTimeseriesLimit(r *http.Request) (int, error) {
	limit, err := QueryParam(r, "limit", strconv.Atoi, Min(1), Max(100))
	if errors.Is(err, ErrParamNotDefined) {
		return defaultTimeseriesLimit, nil
	} else if err != nil {
		return 0, err
	}
	return limit, nil
}

// parseTimeseriesRange parses and validates the before, after and filter query parameters.
// All returned errors are caused by invalid client input and should be answered with
// status 400.
func parseTimeseriesRange(r *http.Request) (timeseriesRange, error) {
	before, err := QueryParam(r, "before", ParseTimeFunc(time.RFC3339Nano))
	if err != nil && !errors.Is(err, ErrParamNotDefined) {
		return timeseriesRange{}, err
	}
	after, err := QueryParam(r, "after", ParseTimeFunc(time.RFC3339Nano))
	if err != nil && !errors.Is(err, ErrParamNotDefined) {
		return timeseriesRange{}, err
	}
	filter := r.FormValue("filter")
	if filter != "" {
		if _, err := validation.ParseRegex(filter, maxFilterLength); err != nil {
			return timeseriesRange{}, err
		}
	}
	return timeseriesRange{Before: before, After: after, Filter: filter}, nil
}

var errRollback = errors.New("transaction rolled back")

// runTx runs f in a transaction. A Responder returned from f rolls it back and answers the request,
// so f rejects a request the way a handler does. Nil for both means f succeeded and it committed.
func runTx(ctx context.Context, f func(ctx context.Context) (respond.Responder, error)) (respond.Responder, error) {
	return runTxWith(ctx, db.RunTx, f)
}

// runTxRR is runTx with isolation level RepeatableRead.
func runTxRR(ctx context.Context, f func(ctx context.Context) (respond.Responder, error)) (respond.Responder, error) {
	return runTxWith(ctx, db.RunTxRR, f)
}

func runTxWith(
	ctx context.Context,
	run func(ctx context.Context, f func(ctx context.Context) error) error,
	f func(ctx context.Context) (respond.Responder, error),
) (respond.Responder, error) {
	var resp respond.Responder
	if err := run(ctx, func(ctx context.Context) error {
		var err error
		if resp, err = f(ctx); err != nil {
			return err
		} else if resp != nil {
			return errRollback
		} else {
			return nil
		}
	}); err != nil && !errors.Is(err, errRollback) {
		return nil, err
	} else {
		return resp, nil
	}
}

// decodeJSON decodes and trims the body. A body that cannot be decoded is answered by the returned Responder.
func decodeJSON[T any](r *http.Request) (T, respond.Responder) {
	return decodeJSONFrom[T](r.Body, "request body is required")
}

// decodeJSONFormValue is [decodeJSON] for the JSON that a multipart request carries in the form field key
// next to its files.
func decodeJSONFormValue[T any](r *http.Request, key string) (T, respond.Responder) {
	return decodeJSONFrom[T](strings.NewReader(r.FormValue(key)), key+" is required")
}

func decodeJSONFrom[T any](reader io.Reader, missing string) (T, respond.Responder) {
	var t T
	if err := json.NewDecoder(reader).Decode(&t); err != nil {
		if errors.Is(err, io.EOF) {
			return t, respond.BadRequest(missing)
		} else if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return t, respond.Error(http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			return t, respond.BadRequest(err.Error())
		}
	}
	validation.TrimStrings(&t)
	return t, nil
}

// attachment is the Content-Disposition value that makes a browser download the response as filename.
func attachment(filename string) string {
	return fmt.Sprintf("attachment; filename=\"%s\"", filename)
}

// maxMultipartFileSize has to stay well below the request body limit of the API routes (requestSize1MiB in
// internal/routing/routing.go), which counts every file of a request together with the multipart encoding
// around them. An application version is created with up to three of them.
const maxMultipartFileSize = 100 * 1024

// readMultipartFile reads the optional YAML file formKey of at most maxMultipartFileSize from a multipart form.
func readMultipartFile(r *http.Request, formKey string) ([]byte, respond.Responder, error) {
	if file, head, err := r.FormFile(formKey); errors.Is(err, http.ErrMissingFile) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get file %v from upload: %w", formKey, err)
	} else if head.Size > maxMultipartFileSize {
		return nil, respond.Error(http.StatusRequestEntityTooLarge, "file too large (max 100 KiB)"), nil
	} else if err := contenttype.IsYaml(head.Header); err != nil {
		return nil, respond.Error(http.StatusUnsupportedMediaType, err.Error()), nil
	} else if data, err := io.ReadAll(file); err != nil {
		return nil, nil, fmt.Errorf("failed to read file %v from upload: %w", formKey, err)
	} else {
		return data, nil, nil
	}
}
