package notification

import (
	"context"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// sendTimeout bounds a dispatched send, which outlives the request it belongs to.
const sendTimeout = 30 * time.Second

// Dispatch runs send in the background so that the request does not wait for the mail server. Run it
// only after the transaction that created what send announces has been committed.
func Dispatch(ctx context.Context, send func(ctx context.Context) error) {
	log := internalctx.GetLogger(ctx)
	go func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, sendTimeout)
		defer cancel()

		if err := send(ctx); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to dispatch notification", zap.Error(err))
		}
	}(context.WithoutCancel(ctx))
}
