package mailsending

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
	"go.uber.org/zap"
)

func SendUserVerificationMail(ctx context.Context, userAccount types.UserAccount, org types.Organization) error {
	mailer := internalctx.GetMailer(ctx)
	log := internalctx.GetLogger(ctx)

	branding, err := db.GetOrganizationBranding(ctx, org.ID)
	if err != nil && !errors.Is(err, apierrors.ErrNotFound) {
		return fmt.Errorf("failed to get organization branding for verification mail: %w", err)
	}

	owb := types.OrganizationWithBranding{Organization: org, Branding: branding}

	// TODO: Should probably use a different mechanism for invite tokens but for now this should work OK
	if _, token, err := authjwt.GenerateVerificationTokenValidFor(userAccount); err != nil {
		log.Error("could not generate verification token for email verification", zap.Error(err))
		return err
	} else {
		if err := mailer.Send(ctx,
			mailx.To(userAccount.Email),
			mailx.Subject("Verify your Distr account"),
			mailx.HtmlBodyTemplate(mailtemplates.VerifyEmail(userAccount, owb, token)),
		); err != nil {
			log.Error(
				"could not send verification mail",
				zap.Error(err),
				zap.String("user", userAccount.Email),
			)
			return err
		} else {
			log.Info("verification mail has been sent", zap.String("user", userAccount.Email))
			return nil
		}
	}
}
