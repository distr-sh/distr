package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
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
	r.Get("/", respond.Handler(getCustomDomainsHandler)).
		With(option.Description("List the custom domains within the caller's scope: every domain for a " +
			"vendor, one customer's own for a customer, or the domains of the customers assigned to a partner")).
		With(option.Response(http.StatusOK, []api.CustomDomain{}))
	r.Get("/{customDomainId}/verification", respond.Handler(getCustomDomainVerificationHandler)).
		With(option.Description("Get the stored result of the last CNAME check of a custom domain, without " +
			"performing a lookup")).
		With(option.Request(customDomainPathRequest{})).
		With(option.Response(http.StatusOK, api.CustomDomainVerification{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/{customDomainId}/verification", respond.Handler(verifyCustomDomainHandler)).
			With(option.Description("Check whether a custom domain's CNAME record currently points at the " +
				"expected target and store the result. This performs a live DNS lookup and can take seconds")).
			With(option.Request(customDomainPathRequest{})).
			With(option.Response(http.StatusOK, api.CustomDomainVerification{}))
		r.With(middleware.RequireCustomDomainsConfigured).Post("/", respond.Handler(createCustomDomainsHandler)).
			With(option.Description("Register new custom domains for the caller's organization, or for a " +
				"customer named in the request")).
			With(option.Request(api.CreateCustomDomainsRequest{})).
			With(option.Response(http.StatusOK, []api.CustomDomain{}))
		r.Delete("/{customDomainId}", respond.Handler(deleteCustomDomainHandler)).
			With(option.Description("Delete a custom domain")).
			With(option.Request(customDomainPathRequest{}))
	})
}

var customDomainNotFound = respond.NotFound("custom domain not found")

func getCustomDomainsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if customDomains, err := db.GetCustomDomainsForScope(
		ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get custom domains: %w", err)
	} else {
		return respond.JSON(mapping.List(customDomains, mapping.CustomDomainToAPI)), nil
	}
}

func getCustomDomainVerificationHandler(r *http.Request) (respond.Responder, error) {
	if domain, resp, err := customDomainOfCaller(r); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(mapping.CustomDomainVerificationToAPI(*domain, false)), nil
	}
}

func verifyCustomDomainHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	if domain, resp, err := customDomainOfCaller(r); resp != nil || err != nil {
		return resp, err
	} else if updated, inconclusive, err := customdomains.CheckAndStore(ctx, *domain); err != nil {
		return nil, fmt.Errorf("failed to verify custom domain: %w", err)
	} else {
		return respond.JSON(mapping.CustomDomainVerificationToAPI(*updated, inconclusive)), nil
	}
}

func customDomainOfCaller(r *http.Request) (*types.CustomDomain, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("customDomainId")); err != nil {
		return nil, customDomainNotFound, nil
	} else if domain, err := db.GetCustomDomainOfOrganization(
		ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return nil, customDomainNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get custom domain: %w", err)
	} else {
		return domain, nil, nil
	}
}

func createCustomDomainsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	request, resp := decodeJSON[api.CreateCustomDomainsRequest](r)
	if resp != nil {
		return resp, nil
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if customerOrgID, resp, err := customerScopeForWrite(
		ctx, request.CustomerOrganizationID,
	); resp != nil || err != nil {
		return resp, err
	} else {
		customDomains := make([]types.CustomDomain, len(request.Domains))
		for i, domain := range request.Domains {
			if customerOrgID != nil && domain.DomainType != types.DomainTypeCustomerPortal {
				return respond.BadRequest("a customer can only register a customer portal domain"), nil
			} else if isPlatformOwnedDomain(domain.Domain) {
				return respond.BadRequest("this domain is owned by the platform and cannot be registered"), nil
			} else if conflict, err := legacyDomainOwnedByOtherOrg(ctx, domain.Domain, *auth.CurrentOrgID()); err != nil {
				return nil, fmt.Errorf("failed to check legacy branding domains: %w", err)
			} else if conflict {
				return respond.Conflict("this domain is already in use"), nil
			} else {
				customDomains[i] = mapping.CustomDomainToInternal(domain, *auth.CurrentOrgID(), customerOrgID)
			}
		}

		if created, err := db.CreateCustomDomains(ctx, customDomains); errors.Is(err, distrerr.ErrConflict) {
			return respond.Conflict("this domain is already in use"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to create custom domains: %w", err)
		} else {
			return respond.JSON(mapping.List(verifyCreatedDomains(ctx, created), mapping.CustomDomainToAPI)), nil
		}
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

func deleteCustomDomainHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("customDomainId")); err != nil {
		return customDomainNotFound, nil
	} else if err := db.DeleteCustomDomain(
		ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return customDomainNotFound, nil
	} else if errors.Is(err, distrerr.ErrConflict) {
		return respond.Conflict(
			"this domain still has an identity provider configured, please delete it on the " +
				"Custom Domains & Identity Provider tab first"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete custom domain: %w", err)
	} else {
		return respond.NoContent(), nil
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
// instance's own app, controller and registry hosts, and the CNAME target host.
func isPlatformOwnedDomain(domain string) bool {
	platformHosts := []string{
		"distr.sh",
		validation.NormalizeHostname(env.Host()),
		validation.NormalizeHostname(env.RegistryHost()),
	}
	if controllerHost := env.ControllerHost(); controllerHost != nil {
		platformHosts = append(platformHosts, validation.NormalizeHostname(*controllerHost))
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
