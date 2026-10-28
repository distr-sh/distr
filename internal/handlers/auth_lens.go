package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/lens"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/util"
)

// authEnterLensHandler switches the session to the lens of a customer organization of the organization it
// operates in.
func authEnterLensHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	request, resp := decodeJSON[api.AuthEnterLensRequest](r)
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	}

	requested := request.CustomerOrganizationID
	if auth.IsLens() && util.PtrEq(&requested, auth.CurrentCustomerOrgID()) {
		return respond.NoContent(), nil
	} else if ownOrg, resp, err := ownContext(ctx, auth, *auth.CurrentOrgID()); resp != nil || err != nil {
		return resp, err
	} else if own := lens.Own(*ownOrg); lens.IsOwn(own, &requested) {
		return respond.BadRequest("this is your own portal, return to it instead"), nil
	} else if _, err := lens.GetCustomerOrganizationForLens(
		ctx, ownOrg.ID, own, auth.IsSuperAdmin(), requested,
	); errors.Is(err, distrerr.ErrNotFound) {
		return invalidCustomerScope, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get customer organization for lens: %w", err)
	} else {
		lensOrg := *ownOrg
		lensOrg.CustomerOrganizationID = &requested
		lensOrg.PartnerOrganizationID = nil
		if tokenString, err := generateContextToken(auth, lensOrg, true); err != nil {
			return nil, err
		} else {
			return respond.JSON(api.AuthLoginResponse{Token: tokenString}), nil
		}
	}
}
