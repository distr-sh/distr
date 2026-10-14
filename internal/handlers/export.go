package handlers

import (
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/subscription"
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

func noRecords[T any](func(T, error) bool) {}

// logExport streams one line per record as a plain text download. An export without any records
// still produces a downloadable (empty) file.
func logExport[T any](filename string, records iter.Seq2[T, error], line func(T) string) respond.Responder {
	return respond.Writer("text/plain; charset=utf-8", func(w io.Writer) (respond.Responder, error) {
		var count int64
		for record, err := range records {
			if err != nil && count > 0 {
				_, _ = io.WriteString(w, exportTruncationNotice)
				return nil, fmt.Errorf("failed to export log records: %w", err)
			} else if errors.Is(err, apierrors.ErrBadRequest) {
				return respond.BadRequest(err.Error()), nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to export log records: %w", err)
			} else if _, err := io.WriteString(w, line(record)); err != nil {
				return nil, fmt.Errorf("failed to write export: %w", err)
			}
			count++
		}
		if subscription.MaxLogExportRows.IsReached(count) {
			if _, err := io.WriteString(w, exportLimitNotice); err != nil {
				return nil, fmt.Errorf("failed to write export: %w", err)
			}
		}
		return nil, nil
	}).WithHeader("Content-Disposition", attachment(filename))
}
