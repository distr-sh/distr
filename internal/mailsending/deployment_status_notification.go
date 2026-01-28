package mailsending

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mail"
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
		mail.TextBody(fmt.Sprintf(`Deployment status has changed:
 * Status: error
 * Deployment Target: %v
 * Application: %v
 * Timestamp: %v
 * Message: %v`,
			deploymentTarget.Name,
			deployment.ApplicationName,
			currentStatus.CreatedAt,
			currentStatus.Message)),
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
		mail.TextBody(fmt.Sprintf(`Deployment status is stale:
 * Deployment Target: %v
 * Application: %v
 * Last Timestamp: %v`,
			deploymentTarget.Name,
			deployment.ApplicationName,
			previousStatus.CreatedAt)),
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
		mail.TextBody(fmt.Sprintf(`Deployment status has changed:
 * Status: %v
 * Deployment Target: %v
 * Application: %v
 * Timestamp: %v
 * Message: %v`,
			currentStatus.Type,
			deploymentTarget.Name,
			deployment.ApplicationName,
			currentStatus.CreatedAt,
			currentStatus.Message)),
		mail.To(user.Email),
	)

	return mailer.Send(ctx, mail)
}
