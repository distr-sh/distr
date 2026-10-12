package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// resolveCustomerScopeForWrite is resolveCustomerScope for custom domains and identity providers,
// which a customer only gets together through its oidc_providers feature. The router already
// enforces the caller's own plan feature.
func resolveCustomerScopeForWrite(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	customerOrgID, ok := resolveCustomerScope(w, r, requested)
	if !ok {
		return nil, false
	}
	if customerOrgID != nil && !requireCustomerOidcProvidersFeature(w, r, *customerOrgID) {
		return nil, false
	}
	return customerOrgID, true
}

// customerScopeQuery declares the parameter that scopes a request to one customer for the OpenAPI
// spec.
type customerScopeQuery struct {
	CustomerOrganizationID *string `query:"customerOrganizationId"`
}

// resolveCustomerScopeFromQuery reads the requested customer organization from the query parameter.
func resolveCustomerScopeFromQuery(w http.ResponseWriter, r *http.Request) (*uuid.UUID, bool) {
	var requested *uuid.UUID
	if value := r.URL.Query().Get("customerOrganizationId"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return respondInvalidCustomerScope(w)
		}
		requested = &id
	}
	return resolveCustomerScope(w, r, requested)
}

// resolveCustomerScope returns the customer a request operates on, or nil for the vendor's own
// organization. It answers the request and returns false when the caller may not reach it.
func resolveCustomerScope(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if customerOrgID := a.CurrentCustomerOrgID(); customerOrgID != nil {
		if requested != nil && *requested != *customerOrgID {
			return respondInvalidCustomerScope(w)
		}
		return customerOrgID, true
	} else if partnerOrgID := a.CurrentPartnerOrgID(); partnerOrgID != nil {
		if requested == nil {
			http.Error(w, "customer organization ID is required", http.StatusBadRequest)
			return nil, false
		}
		if err := db.ValidateCustomerOrgBelongsToPartnerOrg(ctx, *requested, *partnerOrgID); err != nil {
			return respondInvalidCustomerScope(w)
		}
		return requested, true
	} else {
		if requested == nil {
			return nil, true
		}
		if err := db.ValidateCustomerOrgBelongsToOrg(ctx, *requested, *a.CurrentOrgID()); err != nil {
			return respondInvalidCustomerScope(w)
		}
		return requested, true
	}
}

func respondInvalidCustomerScope(w http.ResponseWriter) (*uuid.UUID, bool) {
	http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
	return nil, false
}

func requireCustomerOidcProvidersFeature(w http.ResponseWriter, r *http.Request, customerOrgID uuid.UUID) bool {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	customerOrg := auth.Authentication.Require(ctx).CurrentCustomerOrg()
	if customerOrg == nil || customerOrg.ID != customerOrgID {
		withUsage, err := db.GetCustomerOrganizationByID(ctx, customerOrgID)
		if err != nil {
			log.Error("failed to get customer organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return false
		}
		customerOrg = &withUsage.CustomerOrganization
	}
	if !customerOrg.HasFeature(types.CustomerOrganizationFeatureOidcProviders) {
		http.Error(w, "this customer is not allowed to configure an identity provider", http.StatusForbidden)
		return false
	}
	return true
}
