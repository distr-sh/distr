package handlers

import (
	"context"
	"fmt"
	"net/http"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"go.uber.org/zap"
)

// loginAppDomainRedirect forwards a login on the default host to the organization's app domain. The target is
// resolved with customdomains.AppDomain and never with AppDomainOrDefault: only a self-service CustomDomain row
// is a domain this instance serves itself (it is what Caddy's on-demand TLS asks for), whereas a legacy
// OrganizationBranding.app_domain is a hostname an organization once configured for links and branding, hosted
// in a way Distr knows nothing about. Sending a freshly issued token there could strand the user.
//
// Only a user who belongs to exactly one organization is forwarded. An app domain belongs to one organization
// and brands the whole product for it, so a member of several would be moved into one of them without having
// asked, and a super admin, who reaches every organization, into an arbitrary one.
func loginAppDomainRedirect(ctx context.Context, r *http.Request, user types.UserAccount, token string) *string {
	log := internalctx.GetLogger(ctx)

	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		log.Warn("could not resolve host for login forwarding", zap.Error(err))
		return nil
	}
	if host.source != portalHostDefault || user.IsSuperAdmin {
		return nil
	}

	organizations, err := db.GetOrganizationsForUser(ctx, user.ID)
	if err != nil {
		log.Warn("could not resolve organizations for login forwarding", zap.Error(err))
		return nil
	}
	if len(organizations) != 1 {
		return nil
	}
	domain := customdomains.AppDomain(ctx, organizations[0].ID)
	if domain == nil {
		return nil
	}
	return new(fmt.Sprintf("%v://%v/login?jwt=%v", env.HostScheme(), *domain, token))
}

func redirectLoginToAppDomain(w http.ResponseWriter, r *http.Request, user types.UserAccount, token string) {
	if redirect := loginAppDomainRedirect(r.Context(), r, user, token); redirect != nil {
		http.Redirect(w, r, *redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r,
		fmt.Sprintf("%v/login?jwt=%v", handlerutil.GetRequestSchemeAndHost(r), token),
		http.StatusFound)
}
