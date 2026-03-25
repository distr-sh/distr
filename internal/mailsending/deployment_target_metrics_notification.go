package mailsending

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/types"
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
	usagePercent int,
) error {
	mailer := internalctx.GetMailer(ctx)
	m := mail.New(
		mail.Subject(getDeploymentTargetMetricsNotificationSubject("Alert", metricType, organization, deploymentTarget)),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentTargetMetricsNotificationAlert(
			deploymentTarget, metricType, diskDevice, diskPath, threshold, usagePercent,
		)),
		mail.To(user.Email),
	)
	return mailer.Send(ctx, m)
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
	usagePercent int,
) error {
	mailer := internalctx.GetMailer(ctx)
	m := mail.New(
		mail.Subject(getDeploymentTargetMetricsNotificationSubject("Resolved", metricType, organization, deploymentTarget)),
		mail.HtmlBodyTemplate(mailtemplates.DeploymentTargetMetricsNotificationResolved(
			deploymentTarget, metricType, diskDevice, diskPath, threshold, usagePercent,
		)),
		mail.To(user.Email),
	)
	return mailer.Send(ctx, m)
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
