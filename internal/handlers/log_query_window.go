package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
)

// logQueryRange is the resolved time range and body filter of a log read or export request.
type logQueryRange struct {
	// Start and End are both inclusive. Start defaults to the start of the subscription's
	// log query window, End defaults to now.
	Start time.Time
	End   time.Time
	// Filter is an optional RE2 body filter, already validated.
	Filter string
	// StartExplicit reports whether the client supplied an "after" parameter. The effective
	// order direction must be resolved from this and not from Start, which is always set.
	StartExplicit bool
}

// IsEmpty reports whether the range cannot contain any records. This happens when a "before"
// cursor is older than the resolved window start, e.g. when paginating past the window
// boundary, and must not be forwarded to the log store as an invalid range.
func (r logQueryRange) IsEmpty() bool {
	return r.End.Before(r.Start)
}

// parseLogQueryRange parses the before, after and filter query parameters of a log read or
// export request and resolves them against the subscription's log query window. All returned
// errors are caused by invalid client input and should be answered with status 400.
func parseLogQueryRange(r *http.Request, subscriptionType types.SubscriptionType) (logQueryRange, error) {
	parsed, err := parseTimeseriesRange(r)
	if err != nil {
		return logQueryRange{}, err
	}

	result := logQueryRange{
		End:           parsed.Before,
		Filter:        parsed.Filter,
		StartExplicit: !parsed.After.IsZero(),
	}
	if result.Start, err = resolveLogQueryStart(subscriptionType, parsed.After); err != nil {
		return logQueryRange{}, err
	}
	if result.End.IsZero() {
		result.End = time.Now()
	}
	return result, nil
}

// resolveLogQueryStart defaults an unset "after" parameter to the start of the
// subscription's log query window and rejects explicit values older than that.
// Callers must resolve the effective order direction from the client-supplied "after"
// before applying this default, otherwise requests without an explicit "after" would
// flip from newest-first to oldest-first.
func resolveLogQueryStart(subscriptionType types.SubscriptionType, after time.Time) (time.Time, error) {
	windowStart := subscription.GetLogQueryWindowStart(subscriptionType)
	if after.IsZero() {
		return windowStart, nil
	}
	// Explicit values get the timezone slack so any timezone's 00:00 of the
	// first day inside the window is accepted.
	if after.Before(windowStart.Add(-subscription.LogQueryWindowTimezoneSlack)) {
		return time.Time{}, apierrors.NewBadRequest(
			fmt.Sprintf("after must not be older than %v", subscription.GetLogQueryWindow(subscriptionType)),
		)
	}
	return after, nil
}
