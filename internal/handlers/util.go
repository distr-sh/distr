package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/distr-sh/distr/internal/contenttype"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

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
		if err := handlerutil.ValidateFilterRegex(filter); err != nil {
			return timeseriesRange{}, err
		}
	}
	return timeseriesRange{Before: before, After: after, Filter: filter}, nil
}

func JsonBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var t T
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	return t, err
}

func RespondJSON(w http.ResponseWriter, data any) {
	RespondJSONWithStatus(w, http.StatusOK, data)
}

func RespondJSONWithStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func SetFileDownloadHeaders(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
}

func readMultipartFile(w http.ResponseWriter, r *http.Request, formKey string) ([]byte, bool) {
	log := internalctx.GetLogger(r.Context())
	if file, head, err := r.FormFile(formKey); err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			log.Error("failed to get file from upload", zap.Error(err))
			sentry.GetHubFromContext(r.Context()).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return nil, false
		} else {
			return nil, true
		}
	} else {
		log.Sugar().Debugf("got file %v with type %v and size %v", head.Filename, head.Header, head.Size)
		// max file size is 100KiB
		if head.Size > 102400 {
			log.Debug("large body was rejected")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			fmt.Fprintln(w, "file too large (max 100 KiB)")
			return nil, false
		} else if err := contenttype.IsYaml(head.Header); err != nil {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			fmt.Fprint(w, html.EscapeString(err.Error()))
			return nil, false
		} else if data, err := io.ReadAll(file); err != nil {
			log.Error("failed to read file from upload", zap.Error(err))
			sentry.GetHubFromContext(r.Context()).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return nil, false
		} else {
			return data, true
		}
	}
}
