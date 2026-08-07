package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
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
	r.Get("/", getCustomOIDCConfigurationsHandler).
		With(option.Description("List the OIDC providers within the caller's scope: every provider for " +
			"a vendor, one customer's own for a customer, or the providers of the customers assigned to a partner")).
		With(option.Response(http.StatusOK, api.CustomOIDCConfigurationsResponse{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", createCustomOIDCConfigurationHandler).
			With(option.Description("Configure a new OIDC provider for the current organization")).
			With(option.Request(api.CustomOIDCConfigurationRequest{})).
			With(option.Response(http.StatusOK, api.CustomOIDCConfiguration{}))
		r.Put("/{customOidcConfigurationId}", updateCustomOIDCConfigurationHandler).
			With(option.Description("Update an OIDC provider configuration")).
			With(option.Request(customOIDCConfigurationRequest{})).
			With(option.Response(http.StatusOK, api.CustomOIDCConfiguration{}))
		r.Delete("/{customOidcConfigurationId}", deleteCustomOIDCConfigurationHandler).
			With(option.Description("Delete an OIDC provider configuration and every identity linked to it")).
			With(option.Request(customOIDCConfigurationPathRequest{}))
		r.Post("/{customOidcConfigurationId}/test", testCustomOIDCConfigurationHandler).
			With(option.Description("Check that the configured issuer serves a usable OpenID configuration")).
			With(option.Request(customOIDCConfigurationPathRequest{}))
	})
}

func getCustomOIDCConfigurationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	customerOrgID, partnerOrgID := auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID()

	configurations, err := db.GetCustomOIDCConfigurations(ctx, orgID, customerOrgID, partnerOrgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	domains, err := customDomainsByID(ctx, orgID, customerOrgID, partnerOrgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	// Only reflects the caller's own scope (the vendor's team, or one customer): once a vendor's list
	// spans every customer, a single flat exclusion warning could no longer point at one clear owner.
	members, err := db.GetOrganizationMembersWithOtherOrganizations(ctx, orgID, customerOrgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	RespondJSON(w, api.CustomOIDCConfigurationsResponse{
		Configurations: mapping.List(configurations,
			func(c types.CustomOIDCConfiguration) api.CustomOIDCConfiguration {
				return mapping.CustomOIDCConfigurationToDTO(c, auth.CurrentOrg().Slug, domains[c.CustomDomainID].Domain)
			}),
		MembersWithOtherOrganizations: members,
	})
}

func createCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	request, err := JsonBody[api.CustomOIDCConfigurationRequest](w, r)
	if err != nil {
		return
	}
	customerOrgID, ok := resolveCustomerScopeForWrite(w, r, request.CustomerOrganizationID)
	if !ok {
		return
	}
	if !validateCustomOIDCConfigurationRequest(w, r, &request, customerOrgID, nil, true) {
		return
	}
	if request.ClientSecret == nil {
		http.Error(w, "clientSecret is required", http.StatusBadRequest)
		return
	}

	configuration := types.CustomOIDCConfiguration{
		OrganizationID:         orgID,
		UpdatedByUserAccountID: new(auth.CurrentUserID()),
		ClientSecret:           *request.ClientSecret,
	}
	applyCustomOIDCConfigurationRequest(&configuration, request)

	issuer, ok := resolveCustomOIDCIssuer(w, r, configuration)
	if !ok {
		return
	}
	configuration.Issuer = issuer

	if err := db.CreateCustomOIDCConfiguration(ctx, &configuration); err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	respondCustomOIDCConfiguration(w, r, configuration, customerOrgID, nil)
}

// updateCustomOIDCConfigurationHandler, deleteCustomOIDCConfigurationHandler and
// testCustomOIDCConfigurationHandler all operate on an existing configuration addressed by id, so the
// caller's own auth scope is enough to authorize them: a vendor may reach any row, a partner any row of
// an assigned customer, and a customer only its own. Unlike create, they take no explicit customer
// target from the request.
func updateCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	customerOrgID, partnerOrgID := auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID()

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	request, err := JsonBody[api.CustomOIDCConfigurationRequest](w, r)
	if err != nil {
		return
	}
	if !validateCustomOIDCConfigurationRequest(w, r, &request, customerOrgID, partnerOrgID, false) {
		return
	}

	existing, err := db.GetCustomOIDCConfigurationOfOrganization(ctx, id, orgID, customerOrgID, partnerOrgID)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}

	configuration := *existing
	configuration.UpdatedByUserAccountID = new(auth.CurrentUserID())
	if request.ClientSecret != nil {
		configuration.ClientSecret = *request.ClientSecret
	}
	applyCustomOIDCConfigurationRequest(&configuration, request)

	if configuration.Issuer != existing.Issuer {
		issuer, ok := resolveCustomOIDCIssuer(w, r, configuration)
		if !ok {
			return
		}
		configuration.Issuer = issuer
	}

	if err := db.UpdateCustomOIDCConfiguration(ctx, &configuration); errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
	} else {
		respondCustomOIDCConfiguration(w, r, configuration, customerOrgID, partnerOrgID)
	}
}

func deleteCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	err = db.DeleteCustomOIDCConfiguration(ctx, id, orgID, auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func testCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	configuration, err := db.GetCustomOIDCConfigurationOfOrganization(
		ctx, id, orgID, auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	if _, err := oidc.Discover(ctx, configuration.Issuer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func applyCustomOIDCConfigurationRequest(
	configuration *types.CustomOIDCConfiguration,
	request api.CustomOIDCConfigurationRequest,
) {
	configuration.CustomDomainID = request.CustomDomainID
	configuration.Name = request.Name
	configuration.Slug = request.Slug
	configuration.Enabled = request.Enabled
	configuration.Issuer = request.Issuer
	configuration.ClientID = request.ClientID
	configuration.Scopes = request.Scopes
	configuration.PKCEEnabled = request.PKCEEnabled
	configuration.SPInitiated = request.SPInitiated
	configuration.CreateUnknownUsers = request.CreateUnknownUsers
	configuration.DefaultUserRole = request.DefaultUserRole
	configuration.AllowedEmailDomains = request.AllowedEmailDomains
}

// enforceExactScope must be true on create: the domain lookup below runs through the caller's own
// broad auth scope (a vendor sees every domain), so without this check a vendor could attach a
// provider to a customer's domain by id alone, without naming that customer in the request and
// therefore without resolveCustomerScopeForWrite ever checking that customer's oidc_providers feature.
// It must be false on update, where customerOrgID/partnerOrgID are already the caller's own broad
// scope by design (a vendor may retarget any of its domains) and the domain is already guaranteed to
// be one the caller may reach, since it comes from the same scope that authorized the existing row.
func validateCustomOIDCConfigurationRequest(
	w http.ResponseWriter,
	r *http.Request,
	request *api.CustomOIDCConfigurationRequest,
	customerOrgID, partnerOrgID *uuid.UUID,
	enforceExactScope bool,
) bool {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	request.Normalize()
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}

	// The login and callback URL of every provider start with the organization slug.
	if slug := auth.CurrentOrg().Slug; slug == nil || *slug == "" {
		http.Error(w, "the organization needs a slug before an OIDC provider can be configured",
			http.StatusBadRequest)
		return false
	}

	// Only the domains of this request's scope are candidates, so a customer cannot attach a provider
	// to the vendor's app domain and the vendor cannot attach one to a customer's domain.
	domains, err := customDomainsByID(ctx, orgID, customerOrgID, partnerOrgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return false
	}
	domain, ok := domains[request.CustomDomainID]
	if !ok {
		http.Error(w, "unknown custom domain", http.StatusBadRequest)
		return false
	}
	if enforceExactScope && !util.PtrEq(domain.CustomerOrganizationID, customerOrgID) {
		http.Error(w, "unknown custom domain", http.StatusBadRequest)
		return false
	}
	// Gates on the destination domain's own customer, so update can't move a provider onto a
	// different reachable customer that lacks oidc_providers (create already implies this).
	if domain.CustomerOrganizationID != nil && !requireCustomerOidcProvidersFeature(w, r, *domain.CustomerOrganizationID) {
		return false
	}
	if domain.Type == types.DomainTypeRegistry {
		http.Error(w, "the OIDC provider must be bound to an app or customer portal domain",
			http.StatusBadRequest)
		return false
	}
	// The shared customer portal domain serves every customer of the vendor, and nothing in an OIDC
	// response says which one a new user belongs to, so there is no organization to provision into.
	if request.CreateUnknownUsers &&
		domain.Type == types.DomainTypeCustomerPortal && domain.CustomerOrganizationID == nil {
		http.Error(w, "users cannot be created automatically on the shared customer portal domain, "+
			"because the provider does not say which customer they belong to", http.StatusBadRequest)
		return false
	}
	return true
}

func resolveCustomOIDCIssuer(
	w http.ResponseWriter,
	r *http.Request,
	configuration types.CustomOIDCConfiguration,
) (string, bool) {
	discovered, err := oidc.Discover(r.Context(), configuration.Issuer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", false
	}
	return discovered.Issuer, true
}

func respondCustomOIDCConfiguration(
	w http.ResponseWriter,
	r *http.Request,
	configuration types.CustomOIDCConfiguration,
	customerOrgID, partnerOrgID *uuid.UUID,
) {
	ctx := r.Context()
	domains, err := customDomainsByID(ctx, configuration.OrganizationID, customerOrgID, partnerOrgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	organizationSlug := auth.Authentication.Require(ctx).CurrentOrg().Slug
	RespondJSON(w, mapping.CustomOIDCConfigurationToDTO(
		configuration, organizationSlug, domains[configuration.CustomDomainID].Domain))
}

func respondCustomOIDCConfigurationError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	switch {
	case errors.Is(err, apierrors.ErrConflict):
		http.Error(w, "a provider with this name already exists, or another one is already the default",
			http.StatusConflict)
	case errors.Is(err, apierrors.ErrBadRequest):
		http.Error(w, "invalid custom OIDC configuration", http.StatusBadRequest)
	default:
		internalctx.GetLogger(ctx).Error("custom OIDC configuration request failed", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
