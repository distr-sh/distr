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
	window := subscription.GetLogQueryWindow(subscriptionType)
	windowStart := time.Now().Add(-window)
	if after.IsZero() {
		return windowStart, nil
	}
	if after.Before(windowStart) {
		return time.Time{}, apierrors.NewBadRequest(
			fmt.Sprintf("after must not be older than %v", window),
		)
	}
	return after, nil
}
