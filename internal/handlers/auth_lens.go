package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/lens"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

// authEnterLensHandler switches the session to the lens of a customer organization of the organization it
// operates in.
func authEnterLensHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if request, resp := decodeJSON[api.AuthEnterLensRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if isAlreadyInLens(auth, request.CustomerOrganizationID) {
		return respond.NoContent(), nil
	} else if lensOrg, resp, err := lensContext(ctx, auth, request.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else {
		return contextTokenResponse(auth, *lensOrg, true)
	}
}

func isAlreadyInLens(auth authinfo.AuthInfo, customerOrgID uuid.UUID) bool {
	return auth.IsLens() && util.PtrEq(&customerOrgID, auth.CurrentCustomerOrgID())
}

// lensContext loads the context of the session in the lens of the customer organization customerOrgID.
func lensContext(
	ctx context.Context,
	auth authinfo.AuthInfo,
	customerOrgID uuid.UUID,
) (*types.OrganizationWithUserRole, respond.Responder, error) {
	if ownOrg, resp, err := ownContext(ctx, auth, *auth.CurrentOrgID()); resp != nil || err != nil {
		return nil, resp, err
	} else if !ownOrg.HasActiveSubscription() {
		return nil, respond.Forbidden("the subscription of your organization has expired"), nil
	} else if resp, err := authorizeLens(ctx, auth, *ownOrg, customerOrgID); resp != nil || err != nil {
		return nil, resp, err
	} else {
		return lensOrganization(*ownOrg, customerOrgID), nil, nil
	}
}

func authorizeLens(
	ctx context.Context,
	auth authinfo.AuthInfo,
	ownOrg types.OrganizationWithUserRole,
	customerOrgID uuid.UUID,
) (respond.Responder, error) {
	own := lens.Own(ownOrg)
	if lens.IsOwn(own, &customerOrgID) {
		return respond.BadRequest("this is your own portal, return to it instead"), nil
	} else if _, err := lens.GetCustomerOrganizationForLens(
		ctx, ownOrg.ID, own, auth.IsSuperAdmin(), customerOrgID,
	); errors.Is(err, distrerr.ErrNotFound) {
		return invalidCustomerScope, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get customer organization for lens: %w", err)
	} else {
		return nil, nil
	}
}

func lensOrganization(org types.OrganizationWithUserRole, customerOrgID uuid.UUID) *types.OrganizationWithUserRole {
	org.CustomerOrganizationID = &customerOrgID
	org.PartnerOrganizationID = nil
	return &org
}
