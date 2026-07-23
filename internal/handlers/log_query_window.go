package handlers

import (
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
)

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
