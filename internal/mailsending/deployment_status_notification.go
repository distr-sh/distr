package mailsending

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
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
	return mailer.Send(ctx,
		mailx.Subject(getDeploymentStatusNotificationSubject(
			"Error",
			organization,
			deploymentTarget,
			deployment,
		)),
		mailx.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationError(
			deploymentTarget,
			deployment,
			currentStatus,
		)),
		mailx.To(user.Email),
	)
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
	return mailer.Send(ctx,
		mailx.Subject(getDeploymentStatusNotificationSubject(
			"Stale",
			organization,
			deploymentTarget,
			deployment,
		)),
		mailx.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationStale(
			deploymentTarget,
			deployment,
			previousStatus,
		)),
		mailx.To(user.Email),
	)
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
	return mailer.Send(ctx,
		mailx.Subject(getDeploymentStatusNotificationSubject(
			"Recovered",
			organization,
			deploymentTarget,
			deployment,
		)),
		mailx.HtmlBodyTemplate(mailtemplates.DeploymentStatusNotificationRecovered(
			deploymentTarget,
			deployment,
			currentStatus,
		)),
		mailx.To(user.Email),
	)
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
		eventType, organization.Name, deployment.Application.Name, deployment.ApplicationVersionName,
		deploymentTargetName)
}
