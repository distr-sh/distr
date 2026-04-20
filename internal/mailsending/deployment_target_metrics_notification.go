package mailsending

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
)

func DeploymentTargetMetricsNotificationAlert(
	ctx context.Context,
	user types.UserAccount,
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
	metricType string,
	diskDevice string,
	diskPath string,
	threshold int,
	usagePercent int64,
) error {
	mailer := internalctx.GetMailer(ctx)
	return mailer.Send(ctx,
		mailx.Subject(getDeploymentTargetMetricsNotificationSubject("Alert", metricType, organization, deploymentTarget)),
		mailx.HtmlBodyTemplate(mailtemplates.DeploymentTargetMetricsNotificationAlert(
			deploymentTarget, metricType, diskDevice, diskPath, threshold, usagePercent,
		)),
		mailx.To(user.Email),
	)
}

func DeploymentTargetMetricsNotificationResolved(
	ctx context.Context,
	user types.UserAccount,
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
	metricType string,
	diskDevice string,
	diskPath string,
	threshold int,
	usagePercent int64,
) error {
	mailer := internalctx.GetMailer(ctx)
	return mailer.Send(ctx,
		mailx.Subject(getDeploymentTargetMetricsNotificationSubject("Resolved", metricType, organization, deploymentTarget)),
		mailx.HtmlBodyTemplate(mailtemplates.DeploymentTargetMetricsNotificationResolved(
			deploymentTarget, metricType, diskDevice, diskPath, threshold, usagePercent,
		)),
		mailx.To(user.Email),
	)
}

func getDeploymentTargetMetricsNotificationSubject(
	eventType string,
	metricType string,
	organization types.Organization,
	deploymentTarget types.DeploymentTargetFull,
) string {
	deploymentTargetName := deploymentTarget.Name
	if deploymentTarget.CustomerOrganization != nil {
		deploymentTargetName = deploymentTarget.CustomerOrganization.Name + " " + deploymentTargetName
	}
	return fmt.Sprintf("[%v] %v: %v usage alert on %v", eventType, organization.Name, metricType, deploymentTargetName)
}
