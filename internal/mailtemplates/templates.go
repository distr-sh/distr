package mailtemplates

import (
	"embed"
	"html/template"
	"io/fs"
	"net/url"
	"path"

	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
)

var (
	//go:embed templates/*
	embeddedFS embed.FS

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

func InviteUser(
	userAccount types.UserAccount,
	organization types.OrganizationWithBranding,
	inviteURL string,
) (*template.Template, any) {
	return templates.Lookup("invite-user.html"),
		map[string]any{
			"UserAccount":  userAccount,
			"Organization": organization,
			"Host":         customdomains.AppDomainOrDefault(organization.Organization),
			"InviteURL":    inviteURL,
		}
}

func InviteCustomer(
	userAccount types.UserAccount,
	organization types.OrganizationWithBranding,
	inviteURL string,
) (*template.Template, any) {
	return templates.Lookup("invite-customer.html"),
		map[string]any{
			"UserAccount":  userAccount,
			"Organization": organization,
			"Host":         customdomains.AppDomainOrDefault(organization.Organization),
			"InviteURL":    inviteURL,
		}
}

func VerifyEmail(userAccount types.UserAccount, org types.Organization, token string) (*template.Template, any) {
	return templates.Lookup("verify-email-registration.html"), map[string]any{
		"UserAccount": userAccount,
		"Host":        customdomains.AppDomainOrDefault(org),
		"Token":       token,
	}
}

func PasswordReset(
	userAccount types.UserAccount,
	organization *types.OrganizationWithBranding,
	token string,
) (*template.Template, any) {
	host := env.Host()
	if organization != nil {
		host = customdomains.AppDomainOrDefault(organization.Organization)
	}
	return templates.Lookup("password-reset.html"), map[string]any{
		"UserAccount":  userAccount,
		"Organization": organization,
		"Host":         host,
		"Token":        token,
	}
}

func UpdateEmail(userAccount types.UserAccount, org types.Organization, token string) (*template.Template, any) {
	return templates.Lookup("update-email.html"), map[string]any{
		"UserAccount": userAccount,
		"Host":        customdomains.AppDomainOrDefault(org),
		"Token":       token,
	}
}

func DeploymentStatusNotificationError(
	deploymentTarget types.DeploymentTarget,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) (*template.Template, any) {
	return deploymentStatusNotification("error", deploymentTarget, deployment, nil, &currentStatus)
}

func DeploymentStatusNotificationStale(
	deploymentTarget types.DeploymentTarget,
	deployment types.DeploymentWithLatestRevision,
	previousStatus types.DeploymentRevisionStatus,
) (*template.Template, any) {
	return deploymentStatusNotification("stale", deploymentTarget, deployment, &previousStatus, nil)
}

func DeploymentStatusNotificationRecovered(
	deploymentTarget types.DeploymentTarget,
	deployment types.DeploymentWithLatestRevision,
	currentStatus types.DeploymentRevisionStatus,
) (*template.Template, any) {
	return deploymentStatusNotification("recovered", deploymentTarget, deployment, nil, &currentStatus)
}

func deploymentStatusNotification(
	eventType string,
	deploymentTarget types.DeploymentTarget,
	deployment types.DeploymentWithLatestRevision,
	previousStatus *types.DeploymentRevisionStatus,
	currentStatus *types.DeploymentRevisionStatus,
) (*template.Template, any) {
	return templates.Lookup("deployment-status-notification.html"), map[string]any{
		"EventType":        eventType,
		"DeploymentTarget": deploymentTarget,
		"Deployment":       deployment,
		"PreviousStatus":   previousStatus,
		"CurrentStatus":    currentStatus,
	}
}
