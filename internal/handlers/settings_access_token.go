package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authkey"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

const (
	accessTokenCredentialsExhaustedMessage = "This access token already has two tokens. " +
		"Delete the one you want to replace before adding another."
	accessTokenLastCredentialMessage = "This access token must keep at least one token. " +
		"Add the replacement first, or delete the access token itself."
	accessTokenRoleExceedsCallerMessage = "token role cannot exceed your own role"
)

var (
	accessTokenNotFound       = respond.NotFound("access token not found")
	accessTokenSecretNotFound = respond.NotFound("access token secret not found")
)

func getAccessTokensHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokens, err := db.GetAccessTokens(ctx, auth.CurrentUserID(), *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("error getting tokens: %w", err)
	} else {
		return respond.JSON(mapping.List(tokens, mapping.AccessTokenToAPI)), nil
	}
}

func createAccessTokenHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if request, resp := decodeJSON[api.CreateAccessTokenRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if resp, err := checkAccessTokenRole(ctx, request.UserRole); resp != nil || err != nil {
		return resp, err
	} else if newToken, err := authkey.NewToken(); err != nil {
		return nil, fmt.Errorf("error creating token: %w", err)
	} else {
		token := types.AccessToken{
			Label:         request.Label,
			UserAccountID: auth.CurrentUserID(),
			Key:           newToken.Key(),
			Secret1: &types.AccessTokenSecret{
				Hash:      newToken.Secret().Hash(),
				ExpiresAt: request.ExpiresAt,
			},
			OrganizationID: *auth.CurrentOrgID(),
			UserRole:       request.UserRole,
		}
		if err := db.CreateAccessToken(ctx, &token); err != nil {
			return nil, fmt.Errorf("error creating token: %w", err)
		} else {
			return respond.JSON(mapping.AccessTokenToAPI(token).WithKey(newToken.Serialize())).
				WithStatus(http.StatusCreated), nil
		}
	}
}

func patchAccessTokenHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokenID, err := uuid.Parse(r.PathValue("accessTokenId")); err != nil {
		return accessTokenNotFound, nil
	} else if patch, resp := decodeJSON[api.PatchAccessTokenRequest](r); resp != nil {
		return resp, nil
	} else if updated, err := db.UpdateAccessTokenLabel(
		ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), patch.Label,
	); errors.Is(err, distrerr.ErrNotFound) {
		return accessTokenNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("error updating token: %w", err)
	} else {
		return respond.JSON(mapping.AccessTokenToAPI(*updated)), nil
	}
}

func deleteAccessTokenHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokenID, err := uuid.Parse(r.PathValue("accessTokenId")); err != nil {
		return accessTokenNotFound, nil
	} else if err := db.DeleteAccessToken(ctx, tokenID, auth.CurrentUserID()); err != nil {
		return nil, fmt.Errorf("error deleting token: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func createAccessTokenSecretHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokenID, err := uuid.Parse(r.PathValue("accessTokenId")); err != nil {
		return accessTokenNotFound, nil
	} else if request, resp := decodeJSON[api.CreateAccessTokenSecretRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if token, err := db.GetAccessToken(
		ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return accessTokenNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	} else if slot := token.FreeSecretSlot(); slot == nil || token.CredentialCount() >= 2 {
		return respond.BadRequest(accessTokenCredentialsExhaustedMessage), nil
	} else if newSecret, err := authkey.NewSecret(); err != nil {
		return nil, fmt.Errorf("error creating token secret: %w", err)
	} else {
		secret := types.AccessTokenSecret{Hash: newSecret.Hash(), ExpiresAt: request.ExpiresAt}
		if updated, err := db.CreateAccessTokenSecret(
			ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), *slot, secret,
		); errors.Is(err, distrerr.ErrConflict) {
			return respond.BadRequest(accessTokenCredentialsExhaustedMessage), nil
		} else if err != nil {
			return nil, fmt.Errorf("error creating token secret: %w", err)
		} else {
			rotated := authkey.NewSecureToken(updated.Key, newSecret)
			return respond.JSON(mapping.AccessTokenToAPI(*updated).WithKey(rotated.Serialize())).
				WithStatus(http.StatusCreated), nil
		}
	}
}

func deleteAccessTokenSecretHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokenID, err := uuid.Parse(r.PathValue("accessTokenId")); err != nil {
		return accessTokenNotFound, nil
	} else if slotValue, err := strconv.Atoi(r.PathValue("slot")); err != nil {
		return accessTokenSecretNotFound, nil
	} else if slot := types.AccessTokenSecretSlot(slotValue); !slot.Valid() {
		return accessTokenSecretNotFound, nil
	} else if token, err := db.GetAccessToken(
		ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return accessTokenNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	} else if token.Secret(slot) == nil {
		return accessTokenSecretNotFound, nil
	} else if token.CredentialCount() < 2 {
		return respond.BadRequest(accessTokenLastCredentialMessage), nil
	} else if err := db.DeleteAccessTokenSecret(
		ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), slot,
	); errors.Is(err, distrerr.ErrConflict) {
		return respond.BadRequest(accessTokenLastCredentialMessage), nil
	} else if err != nil {
		return nil, fmt.Errorf("error deleting token secret: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

// deleteAccessTokenKeyHandler retires the key as a credential of its own, which is the last step of
// migrating a token issued before secrets existed.
func deleteAccessTokenKeyHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokenID, err := uuid.Parse(r.PathValue("accessTokenId")); err != nil {
		return accessTokenNotFound, nil
	} else if token, err := db.GetAccessToken(
		ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) || (err == nil && !token.KeyIsCredential) {
		return accessTokenNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	} else if !token.HasSecrets() {
		return respond.BadRequest(accessTokenLastCredentialMessage), nil
	} else if err := db.RetireAccessTokenKeyCredential(
		ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrConflict) {
		return respond.BadRequest(accessTokenLastCredentialMessage), nil
	} else if err != nil {
		return nil, fmt.Errorf("error retiring token key: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func checkAccessTokenRole(ctx context.Context, requested *types.UserRole) (respond.Responder, error) {
	auth := auth.Authentication.Require(ctx)
	if membershipRole, err := db.GetUserRoleInOrganization(
		ctx, auth.CurrentUserID(), *auth.CurrentOrgID(),
	); err != nil {
		return nil, fmt.Errorf("error getting user role: %w", err)
	} else if !types.AccessTokenRoleAllowed(requested, auth.CurrentUserRole(), membershipRole) {
		return respond.BadRequest(accessTokenRoleExceedsCallerMessage), nil
	} else {
		return nil, nil
	}
}
