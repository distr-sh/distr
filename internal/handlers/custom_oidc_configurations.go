package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

type customOIDCConfigurationPathRequest struct {
	CustomOIDCConfigurationID uuid.UUID `path:"customOidcConfigurationId"`
}

type customOIDCConfigurationRequest struct {
	customOIDCConfigurationPathRequest
	api.CustomOIDCConfigurationRequest
}

func CustomOIDCConfigurationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Custom OIDC Providers"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", respond.Handler(getCustomOIDCConfigurationsHandler)).
		With(option.Description("List the OIDC providers within the caller's scope: every provider for " +
			"a vendor, one customer's own for a customer, or the providers of the customers assigned to a partner")).
		With(option.Response(http.StatusOK, api.CustomOIDCConfigurationsResponse{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", respond.Handler(createCustomOIDCConfigurationHandler)).
			With(option.Description("Configure a new OIDC provider for the current organization")).
			With(option.Request(api.CustomOIDCConfigurationRequest{})).
			With(option.Response(http.StatusOK, api.CustomOIDCConfiguration{}))
		r.Put("/{customOidcConfigurationId}", respond.Handler(updateCustomOIDCConfigurationHandler)).
			With(option.Description("Update an OIDC provider configuration")).
			With(option.Request(customOIDCConfigurationRequest{})).
			With(option.Response(http.StatusOK, api.CustomOIDCConfiguration{}))
		r.Delete("/{customOidcConfigurationId}", respond.Handler(deleteCustomOIDCConfigurationHandler)).
			With(option.Description("Delete an OIDC provider configuration and every identity linked to it")).
			With(option.Request(customOIDCConfigurationPathRequest{}))
		r.Post("/{customOidcConfigurationId}/test", respond.Handler(testCustomOIDCConfigurationHandler)).
			With(option.Description("Check that the configured issuer serves a usable OpenID configuration")).
			With(option.Request(customOIDCConfigurationPathRequest{}))
	})
}

func getCustomOIDCConfigurationsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	customerOrgID, partnerOrgID := auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID()

	if configurations, err := db.GetCustomOIDCConfigurations(ctx, orgID, customerOrgID, partnerOrgID); err != nil {
		return customOIDCConfigurationError(err)
	} else if domains, err := customDomainsByID(ctx, orgID, customerOrgID, partnerOrgID); err != nil {
		return customOIDCConfigurationError(err)
	} else {
		return respond.JSON(api.CustomOIDCConfigurationsResponse{
			Configurations: mapping.List(configurations,
				func(c types.CustomOIDCConfiguration) api.CustomOIDCConfiguration {
					return mapping.CustomOIDCConfigurationToAPI(c, auth.CurrentOrg().Slug, domains[c.CustomDomainID].Domain)
				}),
		}), nil
	}
}

func createCustomOIDCConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	if request, resp := decodeJSON[api.CustomOIDCConfigurationRequest](r); resp != nil {
		return resp, nil
	} else if customerOrgID, resp, err := customerScopeForWrite(
		ctx, request.CustomerOrganizationID,
	); resp != nil || err != nil {
		return resp, err
	} else if resp, err := validateCustomOIDCConfigurationRequest(
		r, &request, customerOrgID, nil, true,
	); resp != nil || err != nil {
		return resp, err
	} else if request.ClientSecret == nil {
		return respond.BadRequest("clientSecret is required"), nil
	} else {
		configuration := types.CustomOIDCConfiguration{
			OrganizationID:         orgID,
			UpdatedByUserAccountID: new(auth.CurrentUserID()),
			ClientSecret:           dbcrypto.String(*request.ClientSecret),
		}
		mapping.CustomOIDCConfigurationToInternal(request, &configuration)

		if issuer, resp := resolveCustomOIDCIssuer(r, configuration); resp != nil {
			return resp, nil
		} else {
			configuration.Issuer = issuer
		}

		if err := db.CreateCustomOIDCConfiguration(ctx, &configuration); err != nil {
			return customOIDCConfigurationError(err)
		} else {
			return customOIDCConfigurationResponse(r, configuration, customerOrgID, nil)
		}
	}
}

// updateCustomOIDCConfigurationHandler, deleteCustomOIDCConfigurationHandler and
// testCustomOIDCConfigurationHandler all operate on an existing configuration addressed by id, so the
// caller's own auth scope is enough to authorize them: a vendor may reach any row, a partner any row of
// an assigned customer, and a customer only its own. Unlike create, they take no explicit customer
// target from the request.
func updateCustomOIDCConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	customerOrgID, partnerOrgID := auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID()

	if id, err := uuid.Parse(r.PathValue("customOidcConfigurationId")); err != nil {
		return customOIDCConfigurationNotFound, nil
	} else if request, resp := decodeJSON[api.CustomOIDCConfigurationRequest](r); resp != nil {
		return resp, nil
	} else if resp, err := validateCustomOIDCConfigurationRequest(
		r, &request, customerOrgID, partnerOrgID, false,
	); resp != nil || err != nil {
		return resp, err
	} else if existing, err := db.GetCustomOIDCConfigurationOfOrganization(
		ctx, id, orgID, customerOrgID, partnerOrgID,
	); errors.Is(err, distrerr.ErrNotFound) {
		return customOIDCConfigurationNotFound, nil
	} else if err != nil {
		return customOIDCConfigurationError(err)
	} else {
		configuration := *existing
		configuration.UpdatedByUserAccountID = new(auth.CurrentUserID())
		if request.ClientSecret != nil {
			configuration.ClientSecret = dbcrypto.String(*request.ClientSecret)
		}
		mapping.CustomOIDCConfigurationToInternal(request, &configuration)

		if configuration.Issuer != existing.Issuer {
			if issuer, resp := resolveCustomOIDCIssuer(r, configuration); resp != nil {
				return resp, nil
			} else {
				configuration.Issuer = issuer
			}
		}

		if err := db.UpdateCustomOIDCConfiguration(ctx, &configuration); errors.Is(err, distrerr.ErrNotFound) {
			return customOIDCConfigurationNotFound, nil
		} else if err != nil {
			return customOIDCConfigurationError(err)
		} else {
			return customOIDCConfigurationResponse(r, configuration, customerOrgID, partnerOrgID)
		}
	}
}

func deleteCustomOIDCConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	if id, err := uuid.Parse(r.PathValue("customOidcConfigurationId")); err != nil {
		return customOIDCConfigurationNotFound, nil
	} else if err := db.DeleteCustomOIDCConfiguration(
		ctx, id, orgID, auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return customOIDCConfigurationNotFound, nil
	} else if err != nil {
		return customOIDCConfigurationError(err)
	} else {
		return respond.NoContent(), nil
	}
}

func testCustomOIDCConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	if id, err := uuid.Parse(r.PathValue("customOidcConfigurationId")); err != nil {
		return customOIDCConfigurationNotFound, nil
	} else if configuration, err := db.GetCustomOIDCConfigurationOfOrganization(
		ctx, id, orgID, auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return customOIDCConfigurationNotFound, nil
	} else if err != nil {
		return customOIDCConfigurationError(err)
	} else if _, err := oidc.Discover(ctx, configuration.Issuer); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		return respond.NoContent(), nil
	}
}

// enforceExactScope must be true on create: the domain lookup below runs through the caller's own
// broad auth scope (a vendor sees every domain), so without this check a vendor could attach a
// provider to a customer's domain by id alone, without naming that customer in the request and
// therefore without customerScopeForWrite ever checking that customer's oidc_providers feature.
// It must be false on update, where customerOrgID/partnerOrgID are already the caller's own broad
// scope by design (a vendor may retarget any of its domains) and the domain is already guaranteed to
// be one the caller may reach, since it comes from the same scope that authorized the existing row.
func validateCustomOIDCConfigurationRequest(
	r *http.Request,
	request *api.CustomOIDCConfigurationRequest,
	customerOrgID, partnerOrgID *uuid.UUID,
	enforceExactScope bool,
) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	request.Normalize()
	if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if slug := auth.CurrentOrg().Slug; slug == nil || *slug == "" {
		// The login and callback URL of every provider start with the organization slug.
		return respond.BadRequest("the organization needs a slug before an OIDC provider can be configured"), nil
	} else if domains, err := customDomainsByID(ctx, orgID, customerOrgID, partnerOrgID); err != nil {
		return customOIDCConfigurationError(err)
	} else if domain, ok := domains[request.CustomDomainID]; !ok {
		// Only the domains of this request's scope are candidates, so a customer cannot attach a provider
		// to the vendor's app domain and the vendor cannot attach one to a customer's domain.
		return respond.BadRequest("unknown custom domain"), nil
	} else if enforceExactScope && !util.PtrEq(domain.CustomerOrganizationID, customerOrgID) {
		return respond.BadRequest("unknown custom domain"), nil
	} else {
		// Gates on the destination domain's own customer, so update can't move a provider onto a
		// different reachable customer that lacks oidc_providers (create already implies this).
		if domain.CustomerOrganizationID != nil {
			if resp, err := checkCustomerOidcProvidersFeature(ctx, *domain.CustomerOrganizationID); resp != nil || err != nil {
				return resp, err
			}
		}
		if domain.Type == types.DomainTypeRegistry {
			return respond.BadRequest("the OIDC provider must be bound to an app or customer portal domain"), nil
		} else if request.CreateUnknownUsers &&
			domain.Type == types.DomainTypeCustomerPortal && domain.CustomerOrganizationID == nil {
			// The shared customer portal domain serves every customer of the vendor, and nothing in an OIDC
			// response says which one a new user belongs to, so there is no organization to provision into.
			return respond.BadRequest("users cannot be created automatically on the shared customer portal domain, " +
				"because the provider does not say which customer they belong to"), nil
		} else {
			return nil, nil
		}
	}
}

func resolveCustomOIDCIssuer(
	r *http.Request,
	configuration types.CustomOIDCConfiguration,
) (string, respond.Responder) {
	if discovered, err := oidc.Discover(r.Context(), configuration.Issuer); err != nil {
		return "", respond.BadRequest(err.Error())
	} else {
		return discovered.Issuer, nil
	}
}

func customOIDCConfigurationResponse(
	r *http.Request,
	configuration types.CustomOIDCConfiguration,
	customerOrgID, partnerOrgID *uuid.UUID,
) (respond.Responder, error) {
	ctx := r.Context()
	if domains, err := customDomainsByID(ctx, configuration.OrganizationID, customerOrgID, partnerOrgID); err != nil {
		return customOIDCConfigurationError(err)
	} else {
		organizationSlug := auth.Authentication.Require(ctx).CurrentOrg().Slug
		return respond.JSON(mapping.CustomOIDCConfigurationToAPI(
			configuration, organizationSlug, domains[configuration.CustomDomainID].Domain)), nil
	}
}

func customOIDCConfigurationError(err error) (respond.Responder, error) {
	switch {
	case errors.Is(err, distrerr.ErrConflict):
		return respond.Conflict("this domain already has a provider with this name or slug, " +
			"or another one is already its default"), nil
	case errors.Is(err, distrerr.ErrBadRequest):
		return respond.BadRequest("invalid custom OIDC configuration"), nil
	default:
		return nil, fmt.Errorf("custom OIDC configuration request failed: %w", err)
	}
}

func customDomainsByID(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) (map[uuid.UUID]types.CustomDomain, error) {
	domains, err := db.GetCustomDomainsForScope(ctx, orgID, customerOrgID, partnerOrgID)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]types.CustomDomain, len(domains))
	for _, domain := range domains {
		result[domain.ID] = domain
	}
	return result, nil
}
