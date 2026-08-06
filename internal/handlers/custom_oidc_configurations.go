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
	r.Use(middleware.RequireVendor, middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", getCustomOIDCConfigurationsHandler).
		With(option.Description("List the OIDC providers configured by the current organization")).
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

	configurations, err := db.GetCustomOIDCConfigurations(ctx, orgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	domains, err := customDomainsByID(ctx, orgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return
	}
	members, err := db.GetOrganizationMembersWithOtherOrganizations(ctx, orgID)
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
	if !validateCustomOIDCConfigurationRequest(w, r, &request) {
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
	respondCustomOIDCConfiguration(w, r, configuration)
}

func updateCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	request, err := JsonBody[api.CustomOIDCConfigurationRequest](w, r)
	if err != nil {
		return
	}
	if !validateCustomOIDCConfigurationRequest(w, r, &request) {
		return
	}

	existing, err := db.GetCustomOIDCConfigurationOfOrganization(ctx, id, orgID)
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
		respondCustomOIDCConfiguration(w, r, configuration)
	}
}

func deleteCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := *auth.Authentication.Require(ctx).CurrentOrgID()

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := db.DeleteCustomOIDCConfiguration(ctx, id, orgID); errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func testCustomOIDCConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := *auth.Authentication.Require(ctx).CurrentOrgID()

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	configuration, err := db.GetCustomOIDCConfigurationOfOrganization(ctx, id, orgID)
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

func validateCustomOIDCConfigurationRequest(
	w http.ResponseWriter,
	r *http.Request,
	request *api.CustomOIDCConfigurationRequest,
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

	domains, err := customDomainsByID(ctx, orgID)
	if err != nil {
		respondCustomOIDCConfigurationError(w, r, err)
		return false
	}
	domain, ok := domains[request.CustomDomainID]
	if !ok {
		http.Error(w, "unknown custom domain", http.StatusBadRequest)
		return false
	}
	if domain.Type != types.DomainTypeApp {
		http.Error(w, "the OIDC provider must be bound to an app domain", http.StatusBadRequest)
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
) {
	ctx := r.Context()
	domains, err := customDomainsByID(ctx, configuration.OrganizationID)
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

func customDomainsByID(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]types.CustomDomain, error) {
	domains, err := db.GetCustomDomains(ctx, orgID)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]types.CustomDomain, len(domains))
	for _, domain := range domains {
		result[domain.ID] = domain
	}
	return result, nil
}
