package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/google/uuid"
)

func getOIDCIdentitiesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if identities, err := db.GetUserAccountOIDCIdentities(ctx, auth.CurrentUserID()); err != nil {
		return nil, fmt.Errorf("error getting OIDC identities: %w", err)
	} else {
		return respond.JSON(mapping.List(identities, mapping.UserAccountOIDCIdentityToAPI)), nil
	}
}

func deleteOIDCIdentityHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	identityID, err := uuid.Parse(r.PathValue("oidcIdentityId"))
	if err != nil {
		return oidcIdentityNotFound, nil
	}

	auth := auth.Authentication.Require(ctx)
	user := auth.CurrentUser()

	// Accounts created via OIDC have no password, so disconnecting the only identity
	// would leave the user without any way to sign in.
	if len(user.PasswordHash) == 0 {
		if count, err := db.CountUserAccountOIDCIdentities(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("error counting OIDC identities: %w", err)
		} else if count <= 1 {
			return respond.Conflict(
				"this is the only way to sign in to your account. Set a password before disconnecting it",
			), nil
		}
	}

	if err := db.DeleteUserAccountOIDCIdentity(ctx, user.ID, identityID); errors.Is(err, distrerr.ErrNotFound) {
		return oidcIdentityNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("error deleting OIDC identity: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}
