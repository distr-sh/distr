package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

// authSwitchContextHandler switches the session to the user's own context in an organization, which also leaves a
// lens.
func authSwitchContextHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if request, resp := decodeJSON[api.AuthSwitchContextRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if resp := forbidCrossOrganizationSwitch(auth, request.OrganizationID); resp != nil {
		return resp, nil
	} else if isAlreadyInOwnContext(auth, request.OrganizationID) {
		return respond.NoContent(), nil
	} else if ownOrg, resp, err := ownContext(ctx, auth, request.OrganizationID); resp != nil || err != nil {
		return resp, err
	} else if err := rememberLastUsedOrganization(ctx, auth, ownOrg.ID); err != nil {
		return nil, err
	} else {
		return contextTokenResponse(auth, *ownOrg, false)
	}
}

// forbidCrossOrganizationSwitch keeps a session confined to its organization from switching to another one.
func forbidCrossOrganizationSwitch(auth authinfo.AuthInfo, orgID uuid.UUID) respond.Responder {
	if auth.OrganizationScoped() && orgID != *auth.CurrentOrgID() {
		return respond.Forbidden(middleware.CrossOrganizationBlockedMessage)
	}
	return nil
}

func isAlreadyInOwnContext(auth authinfo.AuthInfo, orgID uuid.UUID) bool {
	return orgID == *auth.CurrentOrgID() && !auth.IsLens()
}

func rememberLastUsedOrganization(ctx context.Context, auth authinfo.AuthInfo, orgID uuid.UUID) error {
	if orgID == *auth.CurrentOrgID() {
		return nil
	} else if err := db.UpdateUserAccountLastUsedOrganizationID(ctx, auth.CurrentUserID(), orgID); err != nil {
		return fmt.Errorf("failed to update last used organization ID: %w", err)
	}
	return nil
}

// ownContext loads the user's own context in an organization, regardless of the lens the request is in.
func ownContext(
	ctx context.Context,
	auth authinfo.AuthInfo,
	orgID uuid.UUID,
) (*types.OrganizationWithUserRole, respond.Responder, error) {
	var org *types.OrganizationWithUserRole
	var err error
	if auth.IsSuperAdmin() {
		org, err = superAdminContext(ctx, orgID)
	} else {
		org, err = membershipContext(ctx, auth.CurrentUserID(), orgID)
	}

	if errors.Is(err, distrerr.ErrNotFound) {
		return nil, organizationNotFound, nil
	} else if err != nil {
		return nil, nil, err
	} else {
		return org, nil, nil
	}
}

// superAdminContext loads any organization for a super admin, who has no role, customer or partner organization in
// it.
func superAdminContext(ctx context.Context, orgID uuid.UUID) (*types.OrganizationWithUserRole, error) {
	if org, err := db.GetOrganizationByID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	} else {
		return &types.OrganizationWithUserRole{Organization: *org}, nil
	}
}

func membershipContext(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationWithUserRole, error) {
	if user, org, err := db.GetUserAccountAndOrg(ctx, userID, orgID); err != nil {
		return nil, fmt.Errorf("failed to get organization membership: %w", err)
	} else {
		return &types.OrganizationWithUserRole{
			Organization:           org.Organization,
			UserRole:               user.UserRole,
			CustomerOrganizationID: user.CustomerOrganizationID,
			PartnerOrganizationID:  user.PartnerOrganizationID,
		}, nil
	}
}

func contextTokenResponse(
	auth authinfo.AuthInfoWithUserAndOrganization,
	org types.OrganizationWithUserRole,
	isLens bool,
) (respond.Responder, error) {
	if currentToken, ok := auth.Token().(jwt.Token); !ok {
		return nil, errors.New("failed to generate token: session is not authenticated by a JWT")
	} else if _, tokenString, err := authjwt.GenerateContextToken(
		currentToken, *auth.CurrentUser(), org, isLens,
	); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	} else {
		return respond.JSON(api.AuthLoginResponse{Token: tokenString}), nil
	}
}
