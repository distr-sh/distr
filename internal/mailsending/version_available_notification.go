package mailsending

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
)

func ApplicationUpdateAvailableNotification(
	ctx context.Context,
	recipient types.NotificationRecipient,
	organization types.OrganizationWithBranding,
	application types.NotificationApplication,
	versionName string,
	deployments []types.DeploymentPendingUpdate,
	customerMessage *string,
) error {
	return sendNotificationWithQuota(ctx, organization.ID, recipient.Email,
		mailx.Subject(fmt.Sprintf("[%v] Update available: %v %v",
			organization.Name, application.Name, versionName)),
		mailx.HtmlBodyTemplate(mailtemplates.ApplicationUpdateAvailableNotification(
			ctx, recipient, organization, application, versionName, deployments, customerMessage,
		)),
	)
}

func ArtifactVersionAvailableNotification(
	ctx context.Context,
	recipient types.NotificationRecipient,
	organization types.OrganizationWithBranding,
	artifact types.NotificationArtifact,
	versionName string,
	customerMessage *string,
) error {
	return sendNotificationWithQuota(ctx, organization.ID, recipient.Email,
		mailx.Subject(fmt.Sprintf("[%v] New version available: %v:%v",
			organization.Name, artifact.Name, versionName)),
		mailx.HtmlBodyTemplate(mailtemplates.ArtifactVersionAvailableNotification(
			ctx, recipient, organization, artifact, versionName, customerMessage,
		)),
	)
}
