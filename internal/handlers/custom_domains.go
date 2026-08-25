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
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

type customDomainPathRequest struct {
	CustomDomainID uuid.UUID `path:"customDomainId"`
}

func CustomDomainsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Custom Domains"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", getCustomDomainsHandler).
		With(option.Description("List the custom domains within the caller's scope: every domain for a " +
			"vendor, one customer's own for a customer, or the domains of the customers assigned to a partner")).
		With(option.Response(http.StatusOK, []api.CustomDomain{}))
	r.Get("/{customDomainId}/verification", getCustomDomainVerificationHandler).
		With(option.Description("Get the stored result of the last CNAME check of a custom domain, without " +
			"performing a lookup")).
		With(option.Request(customDomainPathRequest{})).
		With(option.Response(http.StatusOK, api.CustomDomainVerification{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/{customDomainId}/verification", verifyCustomDomainHandler).
			With(option.Description("Check whether a custom domain's CNAME record currently points at the " +
				"expected target and store the result. This performs a live DNS lookup and can take seconds")).
			With(option.Request(customDomainPathRequest{})).
			With(option.Response(http.StatusOK, api.CustomDomainVerification{}))
		r.With(middleware.RequireCustomDomainsConfigured).Post("/", createCustomDomainsHandler).
			With(option.Description("Register new custom domains for the caller's organization, or for a " +
				"customer named in the request")).
			With(option.Request(api.CreateCustomDomainsRequest{})).
			With(option.Response(http.StatusOK, []api.CustomDomain{}))
		r.Delete("/{customDomainId}", deleteCustomDomainHandler).
			With(option.Description("Delete a custom domain")).
			With(option.Request(customDomainPathRequest{}))
	})
}

func getCustomDomainsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	customDomains, err := db.GetCustomDomainsForScope(
		ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if err != nil {
		log.Error("failed to get custom domains", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, mapping.List(customDomains, mapping.CustomDomainToAPI))
}

func getCustomDomainVerificationHandler(w http.ResponseWriter, r *http.Request) {
	domain, ok := customDomainOfCaller(w, r)
	if !ok {
		return
	}
	RespondJSON(w, mapping.CustomDomainVerificationToAPI(*domain, false))
}

func verifyCustomDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain, ok := customDomainOfCaller(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	updated, inconclusive, err := customdomains.CheckAndStore(ctx, *domain)
	if err != nil {
		internalctx.GetLogger(ctx).Error("failed to verify custom domain", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, mapping.CustomDomainVerificationToAPI(*updated, inconclusive))
}

func customDomainOfCaller(w http.ResponseWriter, r *http.Request) (*types.CustomDomain, bool) {
	id, err := uuid.Parse(r.PathValue("customDomainId"))
	if err != nil {
		http.NotFound(w, r)
		return nil, false
	}

	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	domain, err := db.GetCustomDomainOfOrganization(
		ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return nil, false
	} else if err != nil {
		internalctx.GetLogger(ctx).Error("failed to get custom domain", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	return domain, true
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
	customerOrgID, ok := resolveCustomerScopeForWrite(w, r, request.CustomerOrganizationID)
	if !ok {
		return
	}

	customDomains := make([]types.CustomDomain, len(request.Domains))
	for i, domain := range request.Domains {
		if customerOrgID != nil && domain.DomainType != types.DomainTypeCustomerPortal {
			http.Error(w, "a customer can only register a customer portal domain", http.StatusBadRequest)
			return
		}
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
		customDomains[i] = mapping.CustomDomainToInternal(domain, *auth.CurrentOrgID(), customerOrgID)
	}

	if created, err := db.CreateCustomDomains(ctx, customDomains); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "this domain is already in use", http.StatusConflict)
	} else if err != nil {
		log.Error("failed to create custom domains", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(verifyCreatedDomains(ctx, created), mapping.CustomDomainToAPI))
	}
}

func verifyCreatedDomains(ctx context.Context, created []types.CustomDomain) []types.CustomDomain {
	log := internalctx.GetLogger(ctx)
	result := make([]types.CustomDomain, len(created))
	for i, domain := range created {
		if updated, _, err := customdomains.CheckAndStore(ctx, domain); err != nil {
			log.Warn("failed to verify created custom domain", zap.Error(err), zap.String("domain", domain.Domain))
			result[i] = domain
		} else {
			result[i] = *updated
		}
	}
	return result
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

	err = db.DeleteCustomDomain(ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if errors.Is(err, apierrors.ErrConflict) {
		http.Error(w,
			"this domain still has an identity provider configured, please delete it on the "+
				"Custom Domains & Identity Provider tab first",
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
// instance's own app and registry hosts, and the CNAME target host.
func isPlatformOwnedDomain(domain string) bool {
	platformHosts := []string{
		"distr.sh",
		validation.NormalizeHostname(env.Host()),
		validation.NormalizeHostname(env.RegistryHost()),
	}
	if target := env.CustomDomainTarget(); target != nil {
		platformHosts = append(platformHosts, validation.NormalizeHostname(*target))
	}
	for _, host := range platformHosts {
		if host != "" && (domain == host || strings.HasSuffix(domain, "."+host)) {
			return true
		}
	}
	return false
}
