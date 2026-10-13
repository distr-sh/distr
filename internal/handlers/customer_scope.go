package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

var invalidCustomerScope = respond.BadRequest("invalid customer organization ID")

// customerScopeForWrite is customerScope for custom domains and identity providers, which a
// customer only gets together through its oidc_providers feature. The router already enforces the
// caller's own plan feature.
func customerScopeForWrite(ctx context.Context, requested *uuid.UUID) (*uuid.UUID, respond.Responder, error) {
	if customerOrgID, resp, err := customerScope(ctx, requested); resp != nil || err != nil {
		return nil, resp, err
	} else if customerOrgID == nil {
		return nil, nil, nil
	} else if resp, err := checkCustomerOidcProvidersFeature(ctx, *customerOrgID); resp != nil || err != nil {
		return nil, resp, err
	} else {
		return customerOrgID, nil, nil
	}
}

// customerScopeQuery declares the parameter that scopes a request to one customer for the OpenAPI
// spec.
type customerScopeQuery struct {
	CustomerOrganizationID *string `query:"customerOrganizationId"`
}

// customerScopeFromQuery reads the requested customer organization from the query parameter.
func customerScopeFromQuery(r *http.Request) (*uuid.UUID, respond.Responder, error) {
	if value := r.URL.Query().Get("customerOrganizationId"); value == "" {
		return customerScope(r.Context(), nil)
	} else if id, err := uuid.Parse(value); err != nil {
		return nil, invalidCustomerScope, nil
	} else {
		return customerScope(r.Context(), &id)
	}
}

// customerScope returns the customer a request operates on, or nil for the vendor's own
// organization. The returned Responder rejects a customer the caller may not reach.
func customerScope(ctx context.Context, requested *uuid.UUID) (*uuid.UUID, respond.Responder, error) {
	a := auth.Authentication.Require(ctx)

	if customerOrgID := a.CurrentCustomerOrgID(); customerOrgID != nil {
		if requested != nil && *requested != *customerOrgID {
			return nil, invalidCustomerScope, nil
		} else {
			return customerOrgID, nil, nil
		}
	} else if partnerOrgID := a.CurrentPartnerOrgID(); partnerOrgID != nil {
		if requested == nil {
			return nil, respond.BadRequest("customer organization ID is required"), nil
		} else if err := db.ValidateCustomerOrgBelongsToPartnerOrg(
			ctx, *requested, *partnerOrgID,
		); errors.Is(err, db.ErrCustomerOrgNotInPartnerOrg) {
			return nil, invalidCustomerScope, nil
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to validate customer organization scope: %w", err)
		} else {
			return requested, nil, nil
		}
	} else {
		if requested == nil {
			return nil, nil, nil
		} else if err := db.ValidateCustomerOrgBelongsToOrg(
			ctx, *requested, *a.CurrentOrgID(),
		); errors.Is(err, db.ErrCustomerOrgNotInOrg) {
			return nil, invalidCustomerScope, nil
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to validate customer organization scope: %w", err)
		} else {
			return requested, nil, nil
		}
	}
}

func checkCustomerOidcProvidersFeature(ctx context.Context, customerOrgID uuid.UUID) (respond.Responder, error) {
	customerOrg := auth.Authentication.Require(ctx).CurrentCustomerOrg()
	if customerOrg == nil || customerOrg.ID != customerOrgID {
		if withUsage, err := db.GetCustomerOrganizationByID(ctx, customerOrgID); err != nil {
			return nil, fmt.Errorf("failed to get customer organization: %w", err)
		} else {
			customerOrg = &withUsage.CustomerOrganization
		}
	}
	if !customerOrg.HasFeature(types.CustomerOrganizationFeatureOidcProviders) {
		return respond.Forbidden("this customer is not allowed to configure an identity provider"), nil
	} else {
		return nil, nil
	}
}
