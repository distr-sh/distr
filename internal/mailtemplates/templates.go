package mailtemplates

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/url"
	"path"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// logoContentID has to match the "cid:" reference in fragments/logo.html.
const logoContentID = "logo"

var (
	//go:embed templates/*
	embeddedFS embed.FS

	//go:embed images/distr-logo.png
	distrLogoPNG []byte

	templates *template.Template
	funcMap   = template.FuncMap{
		"QueryEscape":    url.QueryEscape,
		"UnsafeHTMLAttr": func(value string) template.HTMLAttr { return template.HTMLAttr(value) },
		"UnsafeHTML":     func(value string) template.HTML { return template.HTML(value) },
		"UnsafeURL":      func(value string) template.URL { return template.URL(value) },
	}
)

func init() {
	if fsys, err := fs.Sub(embeddedFS, "templates"); err != nil {
		panic(err)
	} else {
		templates = template.Must(parse(fsys, "*.html", "fragments/*.html"))
	}
}

func parse(fsys fs.FS, patterns ...string) (*template.Template, error) {
	t := template.New("").Funcs(funcMap)
	for _, p := range patterns {
		if files, err := fs.Glob(fsys, p); err != nil {
			return nil, err
		} else {
			for _, file := range files {
				// funcMap must be present during parsing *and* execution
				if ft, err := template.New("").Funcs(funcMap).ParseFS(fsys, file); err != nil {
					return nil, err
				} else if _, err := t.AddParseTree(file, ft.Lookup(path.Base(file)).Tree); err != nil {
					return nil, err
				}
			}
		}
	}
	return t, nil
}

// htmlBody renders the named template with the Distr logo, which fragments/logo.html references as
// an inline image because Gmail does not display data URLs.
func htmlBody(name string, data map[string]any) mailx.MailOpt {
	return htmlBodyWithLogo(name, data, "Distr", "image/png", distrLogoPNG)
}

// brandedHTMLBody renders the named template with the branding logo of the organization, falling
// back to the Distr logo.
func brandedHTMLBody(
	ctx context.Context,
	name string,
	data map[string]any,
	organization *types.OrganizationWithBranding,
) mailx.MailOpt {
	if organization != nil && organization.Branding != nil && organization.Branding.LogoImageID != nil {
		logoID := *organization.Branding.LogoImageID
		if file, err := db.GetFileWithID(ctx, logoID); err != nil {
			internalctx.GetLogger(ctx).Warn("could not load branding logo for mail, using Distr logo instead",
				zap.String("template", name),
				zap.Stringer("fileId", logoID),
				zap.Error(err))
		} else {
			return htmlBodyWithLogo(name, data, organization.Name, file.ContentType, file.Data)
		}
	}
	return htmlBody(name, data)
}

func htmlBodyWithLogo(name string, data map[string]any, alt, contentType string, logo []byte) mailx.MailOpt {
	data["LogoAlt"] = alt
	return func(mail *mailx.Mail) {
		mailx.HtmlBodyTemplate(templates.Lookup(name), data)(mail)
		mailx.Inline(logoContentID, contentType, logo)(mail)
	}
}

func InviteUser(
	ctx context.Context,
	userAccount types.UserAccount,
	organization types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	invitingUser types.UserAccount,
	targetOrgName string,
	inviteURL string,
) mailx.MailOpt {
	return brandedHTMLBody(ctx, "invite-user.html", map[string]any{
		"UserAccount":   userAccount,
		"Organization":  organization,
		"InvitingUser":  invitingUser,
		"TargetOrgName": targetOrgName,
		"Host": customdomains.CustomerPortalDomainOrDefault(
			ctx, organization.ID, customerOrgID, organization.Branding),
		"InviteURL": inviteURL,
	}, &organization)
}

func VerifyEmail(
	ctx context.Context,
	userAccount types.UserAccount,
	org types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	token string,
	greetWithOrgName bool,
) mailx.MailOpt {
	signature := "Distr"
	if greetWithOrgName && org.Name != "" && org.Name != userAccount.Email {
		signature = org.Name
	}

	return brandedHTMLBody(ctx, "verify-email-registration.html", map[string]any{
		"UserAccount":  userAccount,
		"Organization": org,
		"Signature":    signature,
		"Host":         customdomains.CustomerPortalDomainOrDefault(ctx, org.ID, customerOrgID, org.Branding),
		"Token":        token,
	}, &org)
}

func PasswordReset(
	ctx context.Context,
	userAccount types.UserAccount,
	organization *types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	token string,
) mailx.MailOpt {
	host := env.Host()
	if organization != nil {
		host = customdomains.CustomerPortalDomainOrDefault(
			ctx, organization.ID, customerOrgID, organization.Branding)
	}
	return brandedHTMLBody(ctx, "password-reset.html", map[string]any{
		"UserAccount":  userAccount,
		"Organization": organization,
		"Host":         host,
		"Token":        token,
	}, organization)
}

func UpdateEmail(
	ctx context.Context,
	userAccount types.UserAccount,
	org types.OrganizationWithBranding,
	customerOrgID *uuid.UUID,
	token string,
) mailx.MailOpt {
	return brandedHTMLBody(ctx, "update-email.html", map[string]any{
		"UserAccount":  userAccount,
		"Organization": org,
		"Host":         customdomains.CustomerPortalDomainOrDefault(ctx, org.ID, customerOrgID, org.Branding),
		"Token":        token,
	}, &org)
}

func CustomEmailTest(
	ctx context.Context,
	userAccount types.UserAccount,
	org types.OrganizationWithBranding,
	config types.CustomEmailConfiguration,
) mailx.MailOpt {
	return brandedHTMLBody(ctx, "custom-email-test.html", map[string]any{
		"UserAccount":   userAccount,
		"Organization":  org,
		"Configuration": config,
		// Distr reports the result of the test to the admin, so the organization name must not be
		// used as the signature here.
		"Signature": "Distr",
		"Host":      customdomains.AppDomainOrDefault(ctx, org.ID, org.Branding),
	}, &org)
}

func DeploymentTargetMetricsNotificationAlert(
	deploymentTarget types.DeploymentTargetFull,
	metricType string,
	diskDevice string,
	diskPath string,
	threshold int,
	usagePercent int64,
) mailx.MailOpt {
	return deploymentTargetMetricsNotification(
		false, deploymentTarget, metricType, diskDevice, diskPath, threshold, usagePercent,
	)
}

func DeploymentTargetMetricsNotificationResolved(
	deploymentTarget types.DeploymentTargetFull,
	metricType string,
	diskDevice string,
	diskPath string,
	threshold int,
	usagePercent int64,
) mailx.MailOpt {
	return deploymentTargetMetricsNotification(
		true, deploymentTarget, metricType, diskDevice, diskPath, threshold, usagePercent,
	)
}

func deploymentTargetMetricsNotification(
	resolved bool,
	deploymentTarget types.DeploymentTargetFull,
	metricType string,
	diskDevice string,
	diskPath string,
	threshold int,
	usagePercent int64,
) mailx.MailOpt {
	return htmlBody("deployment-target-metrics-notification.html", map[string]any{
		"Resolved":         resolved,
		"DeploymentTarget": deploymentTarget,
		"MetricType":       metricType,
		"DiskDevice":       diskDevice,
		"DiskPath":         diskPath,
		"Threshold":        threshold,
		"UsagePercent":     usagePercent,
	})
}

func ApplicationUpdateAvailableNotification(
	ctx context.Context,
	recipient types.NotificationRecipient,
	organization types.OrganizationWithBranding,
	application types.NotificationApplication,
	versionName string,
	deployments []types.DeploymentPendingUpdate,
) mailx.MailOpt {
	return brandedHTMLBody(ctx, "application-update-notification.html", map[string]any{
		"UserAccount":   recipient,
		"Organization":  organization,
		"Host":          notificationHost(ctx, organization, recipient),
		"Application":   application,
		"VersionName":   versionName,
		"Deployments":   deployments,
		"ShowCustomers": recipient.CustomerOrganizationID == nil,
	}, &organization)
}

func ArtifactVersionAvailableNotification(
	ctx context.Context,
	recipient types.NotificationRecipient,
	organization types.OrganizationWithBranding,
	artifact types.NotificationArtifact,
	versionName string,
) mailx.MailOpt {
	return brandedHTMLBody(ctx, "artifact-version-notification.html", map[string]any{
		"UserAccount":  recipient,
		"Organization": organization,
		"Host":         notificationHost(ctx, organization, recipient),
		"Artifact":     artifact,
		"VersionName":  versionName,
	}, &organization)
}

// notificationHost resolves the host a recipient reaches this organization on. A vendor or partner
// recipient has no customer organization, in which case the resolver falls back to the app domain.
func notificationHost(
	ctx context.Context,
	organization types.OrganizationWithBranding,
	recipient types.NotificationRecipient,
) string {
	return customdomains.CustomerPortalDomainOrDefault(
		ctx, organization.ID, recipient.CustomerOrganizationID, organization.Branding)
}

func DeploymentStatusNotificationError(
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) mailx.MailOpt {
	return deploymentStatusNotification("error", deploymentTarget, deployment, nil, &currentStatus)
}

func DeploymentStatusNotificationUpdateFailed(
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) mailx.MailOpt {
	return deploymentStatusNotification("update_failed", deploymentTarget, deployment, nil, &currentStatus)
}

func DeploymentStatusNotificationStale(
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	previousStatus types.DeploymentRevisionStatus,
) mailx.MailOpt {
	return deploymentStatusNotification("stale", deploymentTarget, deployment, &previousStatus, nil)
}

func DeploymentStatusNotificationRecovered(
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) mailx.MailOpt {
	return deploymentStatusNotification("recovered", deploymentTarget, deployment, nil, &currentStatus)
}

func LicenseKeyRevisedCustomer(
	licenseKey types.LicenseKey,
	revision types.LicenseKeyRevision,
	payloadFormatted string,
	token string,
) mailx.MailOpt {
	return htmlBody("license-key-revised-customer.html", map[string]any{
		"LicenseKey":       licenseKey,
		"Revision":         revision,
		"PayloadFormatted": payloadFormatted,
		"Token":            token,
	})
}

func LicenseKeyRevisedVendor(
	licenseKey types.LicenseKey,
	revision types.LicenseKeyRevision,
	payloadFormatted string,
	customerOrgName string,
) mailx.MailOpt {
	return htmlBody("license-key-revised-vendor.html", map[string]any{
		"LicenseKey":       licenseKey,
		"Revision":         revision,
		"PayloadFormatted": payloadFormatted,
		"CustomerOrgName":  customerOrgName,
	})
}

func deploymentStatusNotification(
	eventType string,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	previousStatus *types.DeploymentRevisionStatus,
	currentStatus *types.DeploymentRevisionStatus,
) mailx.MailOpt {
	return htmlBody("deployment-status-notification.html", map[string]any{
		"EventType":        eventType,
		"DeploymentTarget": deploymentTarget,
		"Deployment":       deployment,
		"PreviousStatus":   previousStatus,
		"CurrentStatus":    currentStatus,
	})
}
