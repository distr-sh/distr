package mailsending

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/go-mailx/mailx"
	"go.uber.org/zap"
)

var ErrEmailQuotaExceeded = errors.New("email sending quota exceeded")

// sendNotificationWithQuota sends a notification email to the given address, enforcing the
// hourly per-address quota. If the quota is exhausted, the email is not sent and
// ErrEmailQuotaExceeded is returned. If sending fails, the claimed quota slot is released
// again so that failed sends do not consume quota.
func sendNotificationWithQuota(ctx context.Context, email string, opts ...mailx.MailOpt) error {
	mailer := internalctx.GetMailer(ctx)
	quota := env.NotificationEmailHourlyQuota()
	if quota <= 0 {
		return mailer.Send(ctx, append(opts, mailx.To(email))...)
	}

	recordID, err := db.TryClaimNotificationEmailQuota(ctx, email, quota)
	if err != nil {
		return err
	} else if recordID == nil {
		return fmt.Errorf("%w for %v", ErrEmailQuotaExceeded, email)
	}

	if err := mailer.Send(ctx, append(opts, mailx.To(email))...); err != nil {
		if deleteErr := db.DeleteNotificationEmailRecord(ctx, *recordID); deleteErr != nil {
			internalctx.GetLogger(ctx).Warn("failed to release notification email quota after send error",
				zap.Stringer("recordId", recordID),
				zap.Error(deleteErr),
			)
		}
		return err
	}

	return nil
}
