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
	} else if request.OrganizationID == uuid.Nil {
		return respond.BadRequest("bad request"), nil
	} else if auth.OrganizationScoped() && request.OrganizationID != *auth.CurrentOrgID() {
		return respond.Forbidden(middleware.CrossOrganizationBlockedMessage), nil
	} else if orgChanged := request.OrganizationID != *auth.CurrentOrgID(); !orgChanged && !auth.IsLens() {
		return respond.NoContent(), nil
	} else if ownOrg, resp, err := ownContext(ctx, auth, request.OrganizationID); resp != nil || err != nil {
		return resp, err
	} else if tokenString, err := generateContextToken(auth, *ownOrg, false); err != nil {
		return nil, err
	} else if !orgChanged {
		return respond.JSON(api.AuthLoginResponse{Token: tokenString}), nil
	} else if err := db.UpdateUserAccountLastUsedOrganizationID(ctx, auth.CurrentUserID(), ownOrg.ID); err != nil {
		return nil, fmt.Errorf("failed to update last used organization ID: %w", err)
	} else {
		return respond.JSON(api.AuthLoginResponse{Token: tokenString}), nil
	}
}

// ownContext loads the user's own context in an organization: the organization with the role and the customer or
// partner organization of the user's membership there, regardless of the lens the request is in. Super admins can
// access any organization, where they have neither.
func ownContext(
	ctx context.Context,
	auth authinfo.AuthInfo,
	orgID uuid.UUID,
) (*types.OrganizationWithUserRole, respond.Responder, error) {
	if auth.IsSuperAdmin() {
		if org, err := db.GetOrganizationByID(ctx, orgID); errors.Is(err, distrerr.ErrNotFound) {
			return nil, organizationNotFound, nil
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to get organization: %w", err)
		} else {
			return &types.OrganizationWithUserRole{Organization: *org}, nil, nil
		}
	} else if user, org, err := db.GetUserAccountAndOrg(ctx, auth.CurrentUserID(), orgID); errors.Is(
		err, distrerr.ErrNotFound,
	) {
		return nil, organizationNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get organization membership: %w", err)
	} else {
		return &types.OrganizationWithUserRole{
			Organization:           org.Organization,
			UserRole:               user.UserRole,
			CustomerOrganizationID: user.CustomerOrganizationID,
			PartnerOrganizationID:  user.PartnerOrganizationID,
		}, nil, nil
	}
}

func generateContextToken(
	auth authinfo.AuthInfoWithUserAndOrganization,
	org types.OrganizationWithUserRole,
	isLens bool,
) (string, error) {
	if currentToken, ok := auth.Token().(jwt.Token); !ok {
		return "", errors.New("failed to generate token: session is not authenticated by a JWT")
	} else if _, tokenString, err := authjwt.GenerateContextToken(
		currentToken, *auth.CurrentUser(), org, isLens,
	); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	} else {
		return tokenString, nil
	}
}
