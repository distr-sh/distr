package mailsending

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
)

func DeploymentStatusNotificationError(
	ctx context.Context,
	user types.UserAccount,
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) error {
	mailer := internalctx.GetMailer(ctx)

	mail := mail.New(
		mail.Subject(getDeploymentStatusNotificationSubject(
			"Error",
			organization,
			deploymentTarget,
			deployment,
		)),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationError(
			deploymentTarget,
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
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	previousStatus types.DeploymentRevisionStatus,
) error {
	mailer := internalctx.GetMailer(ctx)

	mail := mail.New(
		mail.Subject(getDeploymentStatusNotificationSubject(
			"Stale",
			organization,
			deploymentTarget,
			deployment,
		)),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationStale(
			deploymentTarget,
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
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) error {
	mailer := internalctx.GetMailer(ctx)

	mail := mail.New(
		mail.Subject(getDeploymentStatusNotificationSubject(
			"Recovered",
			organization,
			deploymentTarget,
			deployment,
		)),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationRecovered(
			deploymentTarget,
			deployment,
			currentStatus,
		)),
		mail.To(user.Email),
	)

	return mailer.Send(ctx, mail)
}

func getDeploymentStatusNotificationSubject(eventType string,
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
) string {
	deploymentTargetName := deploymentTarget.Name
	if deploymentTarget.CustomerOrganization != nil {
		deploymentTargetName = deploymentTarget.CustomerOrganization.Name + " " + deploymentTargetName
	}

	return fmt.Sprintf("[%v] %v deployment %v@%v (%v)",
		eventType, organization.Name, deployment.ApplicationName, deployment.ApplicationVersionName,
		deploymentTargetName)
}
