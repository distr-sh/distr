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
// caller may only ever target itself. Targeting another customer additionally requires that customer's
// own oidc_providers feature, since a customer's domain and identity provider are one bundle gated by
// that single feature; the caller organization's own custom_oidc_providers plan feature is already
// enforced by the router (CustomDomainsFeatureMiddleware / CustomOidcProvidersFeatureMiddleware).
func resolveCustomerScopeForWrite(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	if customerOrgID := a.CurrentCustomerOrgID(); customerOrgID != nil {
		if requested != nil && *requested != *customerOrgID {
			http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
			return nil, false
		}
		return customerOrgID, true
	}

	// A partner never owns vendor-scoped resources: unlike a vendor, "no target" cannot fall back to
	// the caller's own organization, or a partner admin could create app/registry/portal domains and
	// providers directly on the vendor org. Secrets and entitlements enforce the same rule.
	partnerOrgID := a.CurrentPartnerOrgID()
	if requested == nil {
		if partnerOrgID != nil {
			http.Error(w, "customer organization ID is required", http.StatusBadRequest)
			return nil, false
		}
		return nil, true
	}

	if partnerOrgID != nil {
		if err := db.ValidateCustomerOrgBelongsToPartnerOrg(ctx, *requested, *partnerOrgID); err != nil {
			http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
			return nil, false
		}
	} else if err := db.ValidateCustomerOrgBelongsToOrg(ctx, *requested, *a.CurrentOrgID()); err != nil {
		http.Error(w, "invalid customer organization ID", http.StatusBadRequest)
		return nil, false
	}

	customerOrg, err := db.GetCustomerOrganizationByID(ctx, *requested)
	if err != nil {
		log.Error("failed to get customer organization", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, false
	}
	if !customerOrg.HasFeature(types.CustomerOrganizationFeatureOidcProviders) {
		http.Error(w, "this customer is not allowed to configure an identity provider", http.StatusForbidden)
		return nil, false
	}
	return requested, true
}
