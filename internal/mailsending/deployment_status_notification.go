package mailsending

import (
	"context"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
)

func DeploymentStatusNotificationError(
	ctx context.Context,
	user types.UserAccount,
	deploymentTarget types.DeploymentTargetWithCreatedBy,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) error {
	mailer := internalctx.GetMailer(ctx)

	mail := mail.New(
		mail.Subject("Deployment Status Notification [Error]"),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationError(
			deploymentTarget.DeploymentTarget,
			deployment,
			currentStatus,
		)),
		mail.To(user.Email),
	)

	return mailer.Send(ctx, mail)
}

func DeploymentStatusNotificationStale(
	ctx context.Context,
	user types.UserAccount,
	deploymentTarget types.DeploymentTargetWithCreatedBy,
	deployment types.DeploymentWithLatestRevision,
	previousStatus types.DeploymentRevisionStatus,
) error {
	mailer := internalctx.GetMailer(ctx)

	mail := mail.New(
		mail.Subject("Deployment Status Notification [Stale]"),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationStale(
			deploymentTarget.DeploymentTarget,
			deployment,
			previousStatus,
		)),
		mail.To(user.Email),
	)

	return mailer.Send(ctx, mail)
}

func DeploymentStatusNotificationRecovered(
	ctx context.Context,
	user types.UserAccount,
	deploymentTarget types.DeploymentTargetWithCreatedBy,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) error {
	mailer := internalctx.GetMailer(ctx)

	mail := mail.New(
		mail.Subject("Deployment Status Notification [Recovered]"),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationRecovered(
			deploymentTarget.DeploymentTarget,
			deployment,
			currentStatus,
		)),
		mail.To(user.Email),
	)

	return mailer.Send(ctx, mail)
}
