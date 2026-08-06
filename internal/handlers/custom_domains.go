package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func CustomDomainsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Custom Domains"))
	r.Use(middleware.RequireVendor, middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", getCustomDomainsHandler).
		With(option.Description("List all custom domains of the current organization")).
		With(option.Response(http.StatusOK, []types.CustomDomain{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.With(middleware.RequireCustomDomainsConfigured).Post("/", createCustomDomainsHandler).
			With(option.Description("Register new custom domains for the current organization")).
			With(option.Request(api.CreateCustomDomainsRequest{})).
			With(option.Response(http.StatusOK, []types.CustomDomain{}))
		r.Delete("/{customDomainId}", deleteCustomDomainHandler).
			With(option.Description("Delete a custom domain")).
			With(option.Request(struct {
				CustomDomainID uuid.UUID `path:"customDomainId"`
			}{}))
	})
}

func getCustomDomainsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	customDomains, err := db.GetCustomDomains(ctx, *auth.CurrentOrgID())
	if err != nil {
		log.Error("failed to get custom domains", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, customDomains)
}

func createCustomDomainsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	request, err := JsonBody[api.CreateCustomDomainsRequest](w, r)
	if err != nil {
		return
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	customDomains := make([]types.CustomDomain, len(request.Domains))
	for i, domain := range request.Domains {
		if isPlatformOwnedDomain(domain.Domain) {
			http.Error(w, "this domain is owned by the platform and cannot be registered", http.StatusBadRequest)
			return
		}
		if conflict, err := legacyDomainOwnedByOtherOrg(ctx, domain.Domain, *auth.CurrentOrgID()); err != nil {
			log.Error("failed to check legacy branding domains", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if conflict {
			http.Error(w, "this domain is already in use", http.StatusConflict)
			return
		}
		customDomains[i] = types.CustomDomain{
			Domain:         domain.Domain,
			Type:           domain.DomainType,
			OrganizationID: *auth.CurrentOrgID(),
		}
	}

	if created, err := db.CreateCustomDomains(ctx, customDomains); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "this domain is already in use", http.StatusConflict)
	} else if err != nil {
		log.Error("failed to create custom domains", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, created)
	}
}

func deleteCustomDomainHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("customDomainId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if err := db.DeleteCustomDomain(ctx, id, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if errors.Is(err, apierrors.ErrConflict) {
		http.Error(w,
			"this domain still has an identity provider configured, please delete it on the "+
				"Identity Provider tab first",
			http.StatusConflict)
	} else if err != nil {
		log.Error("failed to delete custom domain", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// legacyDomainOwnedByOtherOrg reports whether the given normalized domain is already configured as a
// legacy OrganizationBranding.app_domain / registry_domain of a different organization. Custom domains
// take precedence over the legacy columns during host resolution (portal branding, TLS ask), so
// registering such a domain would let one organization take over another's existing domain. The owning
// organization registering its own legacy domain is fine (self-service migration). Legacy values may
// contain a scheme and/or port, so they are normalized before comparison.
func legacyDomainOwnedByOtherOrg(ctx context.Context, domain string, orgID uuid.UUID) (bool, error) {
	legacyDomains, err := db.GetOrganizationLegacyBrandingDomains(ctx)
	if err != nil {
		return false, err
	}
	for _, legacy := range legacyDomains {
		if legacy.OrganizationID == orgID {
			continue
		}
		for _, legacyDomain := range []*string{legacy.AppDomain, legacy.RegistryDomain} {
			if legacyDomain != nil && validation.NormalizeHostname(*legacyDomain) == domain {
				return true, nil
			}
		}
	}
	return false, nil
}

// isPlatformOwnedDomain reports whether the given normalized domain is owned by the platform
// and must therefore not be registrable as a custom domain: distr.sh (and subdomains), the
// instance's own app and registry hosts, and the CNAME target hosts.
func isPlatformOwnedDomain(domain string) bool {
	platformHosts := []string{
		"distr.sh",
		validation.NormalizeHostname(env.Host()),
		validation.NormalizeHostname(env.RegistryHost()),
	}
	if target := env.CustomDomainAppCNAMETarget(); target != nil {
		platformHosts = append(platformHosts, validation.NormalizeHostname(*target))
	}
	if target := env.CustomDomainRegistryCNAMETarget(); target != nil {
		platformHosts = append(platformHosts, validation.NormalizeHostname(*target))
	}
	for _, host := range platformHosts {
		if host != "" && (domain == host || strings.HasSuffix(domain, "."+host)) {
			return true
		}
	}
	return false
}
