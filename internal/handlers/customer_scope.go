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

// resolveCustomerScopeForWrite determines which scope a create or update targets: the caller's own
// organization (nil) or, for a vendor or partner naming one explicitly, that customer. A customer
// caller may only ever target itself. Every customer scope, whether it is the caller's own or one
// named by a vendor or partner, additionally requires that customer's own oidc_providers feature,
// since a customer's domain and identity provider are one bundle gated by that single feature; the
// caller organization's own custom_oidc_providers plan feature is already enforced by the router
// (CustomDomainsFeatureMiddleware / CustomOidcProvidersFeatureMiddleware).
func resolveCustomerScopeForWrite(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	a := auth.Authentication.Require(r.Context())

	// A partner never owns vendor-scoped resources: unlike a vendor, "no target" cannot fall back to
	// the caller's own organization, or a partner admin could create app/registry/portal domains and
	// providers directly on the vendor org. Secrets and entitlements enforce the same rule.
	if requested == nil && a.CurrentCustomerOrgID() == nil && a.CurrentPartnerOrgID() != nil {
		http.Error(w, "customer organization ID is required", http.StatusBadRequest)
		return nil, false
	}

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
			http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
			return nil, false
		}
		requested = &id
	}
	return resolveCustomerScope(w, r, requested)
}

// resolveCustomerScope determines which customer a request operates on: the caller's own when they
// belong to one, the requested one when a vendor or partner names it and their own organization
// otherwise. It answers the request and returns false when the caller may not reach the requested
// customer.
func resolveCustomerScope(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	a := auth.Authentication.Require(r.Context())

	if customerOrgID := a.CurrentCustomerOrgID(); customerOrgID != nil {
		if requested != nil && *requested != *customerOrgID {
			http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
			return nil, false
		}
		return customerOrgID, true
	}

	if requested == nil {
		return nil, true
	}

	ctx := r.Context()
	var err error
	if partnerOrgID := a.CurrentPartnerOrgID(); partnerOrgID != nil {
		err = db.ValidateCustomerOrgBelongsToPartnerOrg(ctx, *requested, *partnerOrgID)
	} else {
		err = db.ValidateCustomerOrgBelongsToOrg(ctx, *requested, *a.CurrentOrgID())
	}
	if err != nil {
		http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
		return nil, false
	}
	return requested, true
}

func requireCustomerOidcProvidersFeature(w http.ResponseWriter, r *http.Request, customerOrgID uuid.UUID) bool {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	customerOrg, err := db.GetCustomerOrganizationByID(ctx, customerOrgID)
	if err != nil {
		log.Error("failed to get customer organization", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return false
	}
	if !customerOrg.HasFeature(types.CustomerOrganizationFeatureOidcProviders) {
		http.Error(w, "this customer is not allowed to configure an identity provider", http.StatusForbidden)
		return false
	}
	return true
}
