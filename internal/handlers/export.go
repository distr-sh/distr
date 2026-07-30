package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// exportTruncationNotice is appended to an export when streaming fails midway: the response
// status has already been sent at that point, so the client would otherwise receive a
// silently truncated file.
const exportTruncationNotice = `
##################################################
export possibly truncated due to an internal error
`

// exportLimitNotice is appended to an export that hit [subscription.MaxLogExportRows], so the
// truncation is visible in the downloaded file.
var exportLimitNotice = fmt.Sprintf(`
##################################################
export truncated at the limit of %d lines, select a smaller time range to export all lines
`, subscription.MaxLogExportRows)

// exportWriter streams a plain text export to the client. The download headers are only set
// right before the first line, so an error response can still be sent as long as nothing has
// been written yet.
type exportWriter struct {
	w        http.ResponseWriter
	log      *zap.Logger
	filename string
	written  bool
	count    int64
	writeErr error
}

func newExportWriter(w http.ResponseWriter, log *zap.Logger, filename string) *exportWriter {
	return &exportWriter{w: w, log: log, filename: filename}
}

// writeLine writes a single exported line. A returned error means the client is gone and the
// export must be abandoned; it is already logged.
func (e *exportWriter) writeLine(format string, args ...any) error {
	if !e.written {
		SetFileDownloadHeaders(e.w, e.filename)
		e.written = true
	}
	if _, err := fmt.Fprintf(e.w, format, args...); err != nil {
		e.log.Error("failed to write export to response writer", zap.Error(err))
		e.writeErr = err
		return err
	}
	e.count++
	return nil
}

// fail ends an export that could not read all of its records.
func (e *exportWriter) fail(ctx context.Context, msg string, err error) {
	e.log.Error(msg, zap.Error(err))
	switch {
	case e.writeErr != nil:
		// The export was abandoned because the client is gone, so there is nobody left to
		// respond to.
	case e.written:
		_, _ = e.w.Write([]byte(exportTruncationNotice))
	case errors.Is(err, apierrors.ErrBadRequest):
		http.Error(e.w, err.Error(), http.StatusBadRequest)
	default:
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(e.w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// finish ends an export that read all of its records. An export without any records still
// produces a downloadable (empty) file, and one that hit the row limit is marked as truncated.
func (e *exportWriter) finish() {
	if !e.written {
		SetFileDownloadHeaders(e.w, e.filename)
	} else if subscription.MaxLogExportRows.IsReached(e.count) {
		_, _ = e.w.Write([]byte(exportLimitNotice))
	}
}
