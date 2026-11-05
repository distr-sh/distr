package mailsending

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/custommail"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrInviteMailNotSent wraps the error of a mail server that rejected or never received the
// invite mail, as opposed to a failure to prepare it.
var ErrInviteMailNotSent = errors.New("invite mail could not be sent")

func SendUserInviteMail(
	ctx context.Context,
	userAccount types.UserAccount,
	organization types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	inviteURL string,
) error {
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	mailer, err := custommail.MailerForOrganization(ctx, organization.ID)
	if err != nil {
		return err
	}

	from, err := custommail.FromAddressOrDefault(ctx, organization.ID, organization.Branding)
	if err != nil {
		return err
	}
	from.Name = organization.Name
	subject := "Welcome to Distr"
	if organization.Name != "" {
		subject = "Welcome to " + organization.Name
	}

	currentUser, err := db.GetUserAccountByID(ctx, auth.CurrentUserID())
	if err != nil {
		log.Error("could not get current user for invite mail", zap.Error(err))
		return err
	}

	targetOrgName := organization.Name
	if customerOrgID != nil {
		customerOrg, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID)
		if err != nil {
			log.Error("could not get customer organization for invite mail", zap.Error(err))
			return err
		}
		targetOrgName = customerOrg.Name
	}

	if err := mailer.Send(ctx,
		mailx.To(userAccount.Email),
		mailx.From(*from),
		mailx.ReplyTo(currentUser.Email),
		mailx.Subject(subject),
		mailtemplates.InviteUser(
			ctx, userAccount, organization, customerOrgID, *currentUser, targetOrgName, inviteURL),
	); err != nil {
		log.Warn(
			"could not send invite mail",
			zap.Error(err),
			zap.String("user", userAccount.Email),
		)
		return fmt.Errorf("%w: %w", ErrInviteMailNotSent, err)
	} else {
		log.Info("invite mail has been sent", zap.String("user", userAccount.Email))
		return nil
	}
}
