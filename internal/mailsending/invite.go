package mailsending

import (
	"context"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func SendUserInviteMail(
	ctx context.Context,
	userAccount types.UserAccount,
	organization types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	inviteURL string,
) error {
	mailer := internalctx.GetMailer(ctx)
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	from, err := customdomains.EmailFromAddressParsedOrDefault(organization.Organization)
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

	email := mail.New(
		mail.To(userAccount.Email),
		mail.From(*from),
		mail.Bcc(currentUser.Email),
		mail.ReplyTo(currentUser.Email),
		mail.Subject(subject),
		mail.HtmlBodyTemplate(mailtemplates.InviteUser(userAccount, organization, *currentUser, targetOrgName, inviteURL)),
	)

	if err := mailer.Send(ctx, email); err != nil {
		log.Error(
			"could not send invite mail",
			zap.Error(err),
			zap.String("user", userAccount.Email),
		)
		return err
	} else {
		log.Info("invite mail has been sent", zap.String("user", userAccount.Email))
		return nil
	}
}
