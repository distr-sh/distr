package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

var errInvalidCustomerScope = apierrors.NewBadRequest("invalid customer organization ID")

// customerScopeForWrite is customerScope for custom domains and identity providers, which a
// customer only gets together through its oidc_providers feature. The router already enforces the
// caller's own plan feature.
func customerScopeForWrite(ctx context.Context, requested *uuid.UUID) (*uuid.UUID, error) {
	customerOrgID, err := customerScope(ctx, requested)
	if err != nil {
		return nil, err
	}
	if customerOrgID != nil {
		if err := checkCustomerOidcProvidersFeature(ctx, *customerOrgID); err != nil {
			return nil, err
		}
	}
	return customerOrgID, nil
}

// customerScopeQuery declares the parameter that scopes a request to one customer for the OpenAPI
// spec.
type customerScopeQuery struct {
	CustomerOrganizationID *string `query:"customerOrganizationId"`
}

// customerScopeFromQuery reads the requested customer organization from the query parameter.
func customerScopeFromQuery(r *http.Request) (*uuid.UUID, error) {
	var requested *uuid.UUID
	if value := r.URL.Query().Get("customerOrganizationId"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, errInvalidCustomerScope
		}
		requested = &id
	}
	return customerScope(r.Context(), requested)
}

// customerScope returns the customer a request operates on, or nil for the vendor's own
// organization. It fails when the caller may not reach it.
func customerScope(ctx context.Context, requested *uuid.UUID) (*uuid.UUID, error) {
	a := auth.Authentication.Require(ctx)

	if customerOrgID := a.CurrentCustomerOrgID(); customerOrgID != nil {
		if requested != nil && *requested != *customerOrgID {
			return nil, errInvalidCustomerScope
		}
		return customerOrgID, nil
	} else if partnerOrgID := a.CurrentPartnerOrgID(); partnerOrgID != nil {
		if requested == nil {
			return nil, apierrors.NewBadRequest("customer organization ID is required")
		}
		err := db.ValidateCustomerOrgBelongsToPartnerOrg(ctx, *requested, *partnerOrgID)
		if errors.Is(err, db.ErrCustomerOrgNotInPartnerOrg) {
			return nil, errInvalidCustomerScope
		} else if err != nil {
			return nil, fmt.Errorf("failed to validate customer organization scope: %w", err)
		}
		return requested, nil
	} else {
		if requested == nil {
			return nil, nil
		}
		err := db.ValidateCustomerOrgBelongsToOrg(ctx, *requested, *a.CurrentOrgID())
		if errors.Is(err, db.ErrCustomerOrgNotInOrg) {
			return nil, errInvalidCustomerScope
		} else if err != nil {
			return nil, fmt.Errorf("failed to validate customer organization scope: %w", err)
		}
		return requested, nil
	}
}

func checkCustomerOidcProvidersFeature(ctx context.Context, customerOrgID uuid.UUID) error {
	customerOrg := auth.Authentication.Require(ctx).CurrentCustomerOrg()
	if customerOrg == nil || customerOrg.ID != customerOrgID {
		withUsage, err := db.GetCustomerOrganizationByID(ctx, customerOrgID)
		if err != nil {
			return fmt.Errorf("failed to get customer organization: %w", err)
		}
		customerOrg = &withUsage.CustomerOrganization
	}
	if !customerOrg.HasFeature(types.CustomerOrganizationFeatureOidcProviders) {
		return apierrors.NewForbidden("this customer is not allowed to configure an identity provider")
	}
	return nil
}

func resolveCustomerScopeForWrite(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	id, err := customerScopeForWrite(r.Context(), requested)
	return id, respondUnlessNil(w, r, err)
}

func resolveCustomerScopeFromQuery(w http.ResponseWriter, r *http.Request) (*uuid.UUID, bool) {
	id, err := customerScopeFromQuery(r)
	return id, respondUnlessNil(w, r, err)
}

func resolveCustomerScope(w http.ResponseWriter, r *http.Request, requested *uuid.UUID) (*uuid.UUID, bool) {
	id, err := customerScope(r.Context(), requested)
	return id, respondUnlessNil(w, r, err)
}

func requireCustomerOidcProvidersFeature(w http.ResponseWriter, r *http.Request, customerOrgID uuid.UUID) bool {
	return respondUnlessNil(w, r, checkCustomerOidcProvidersFeature(r.Context(), customerOrgID))
}

// respondUnlessNil answers a non-nil err and reports whether the caller may go on.
func respondUnlessNil(w http.ResponseWriter, r *http.Request, err error) bool {
	if err != nil {
		respond.Error(w, r, err)
		return false
	}
	return true
}
