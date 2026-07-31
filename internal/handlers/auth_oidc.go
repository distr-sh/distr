package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/userauth"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

const (
	redirectToLoginOIDCFailed               = "/login?reason=oidc-failed"
	redirectToLoginOIDCRegistrationDisabled = "/login?reason=oidc-registration-disabled"
)

func AuthOIDCRouter(r chiopenapi.Router) {
	type OIDCRequest struct {
		OIDCProvider string `path:"oidcProvider"`
	}

	r.Get("/{oidcProvider}", authLoginOidcHandler).
		With(option.Request(OIDCRequest{}))
	r.Get("/{oidcProvider}/callback", authLoginOidcCallbackHandler).
		With(option.Request(OIDCRequest{}))
}

func authLoginOidcHandler(w http.ResponseWriter, r *http.Request) {
	provider := oidc.Provider(r.PathValue("oidcProvider"))
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	if state, pkceVerifier, err := db.CreateOIDCState(ctx); err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC state creation failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	} else {
		oidcer := internalctx.GetOIDCer(ctx)
		redirectURL, err := oidcer.GetAuthCodeURL(r, provider, state.String(), pkceVerifier)
		if err != nil {
			http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

func authLoginOidcCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	pkceVerifier, err := verifyOIDCState(r)
	if err != nil {
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
			return
		}

		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Warn("could not verify OIDC state", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	provider := oidc.Provider(r.PathValue("oidcProvider"))
	log = log.With(zap.String("provider", string(provider)))

	if oidcError := r.URL.Query().Get("error"); oidcError != "" {
		log.Warn("OIDC provider returned error",
			zap.String("error", oidcError),
			zap.String("error_description", r.URL.Query().Get("error_description")))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		log.Warn("OIDC callback missing code parameter")
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	oidcer := internalctx.GetOIDCer(ctx)
	identity, err := oidcer.GetIdentityForCode(ctx, provider, code, pkceVerifier, r)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC identity extraction failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		user, err := resolveOIDCUser(ctx, log, identity)
		if err != nil {
			return err
		}
		if user == nil {
			http.Redirect(w, r, redirectToLoginOIDCRegistrationDisabled, http.StatusFound)
			return nil
		}
		log = log.With(zap.Any("userId", user.ID))

		if user.EmailVerifiedAt == nil && identity.EmailVerified {
			if err = db.UpdateUserAccountEmailVerified(ctx, user); err != nil {
				return err
			}
		}
		if tokenString, err := userauth.GenerateLoginToken(ctx, *user); err != nil {
			return fmt.Errorf("token creation failed: %w", err)
		} else if err = db.UpdateUserAccountLastLoggedIn(ctx, user.ID); err != nil {
			return err
		} else {
			http.Redirect(w, r,
				fmt.Sprintf("%v/login?jwt=%v", handlerutil.GetRequestSchemeAndHost(r), tokenString),
				http.StatusFound)
			return nil
		}
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Warn("user login failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
	}
}

// resolveOIDCUser returns the user account the given identity provider identity belongs to.
// The identity is looked up first, so a login keeps working when the email address changed
// on either side. Accounts that predate the identity linking are matched by email and get
// their identity created on the fly. Returns (nil, nil) if the user would have to be
// registered but registration is administratively disabled.
func resolveOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	identity oidc.Identity,
) (*types.UserAccount, error) {
	user, existingIdentity, err := db.GetUserAccountWithOIDCIdentity(ctx, identity.Issuer, identity.Subject)
	if err == nil {
		// The user account email is authoritative and is never overwritten with the one
		// from the identity provider, which is only kept on the identity for display.
		return user, db.UpdateUserAccountOIDCIdentityOnLogin(ctx, existingIdentity.ID, new(identity.Email))
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return nil, err
	}

	user, err = db.GetUserAccountByEmail(ctx, identity.Email)
	if errors.Is(err, apierrors.ErrNotFound) {
		if user, err = registerOIDCUser(ctx, identity.Email); err != nil {
			return nil, err
		} else if user == nil {
			return nil, nil
		}
		log.Info("registered new user account for OIDC identity", zap.Any("userId", user.ID))
	} else if err != nil {
		return nil, err
	} else {
		log.Info("linking OIDC identity to existing user account matched by email",
			zap.Any("userId", user.ID))
	}

	newIdentity := types.UserAccountOIDCIdentity{
		UserAccountID: user.ID,
		Provider:      identity.Provider,
		Issuer:        identity.Issuer,
		Subject:       identity.Subject,
		Email:         new(identity.Email),
	}
	if err := db.CreateUserAccountOIDCIdentity(ctx, &newIdentity); err != nil {
		return nil, err
	}
	return user, nil
}

// registerOIDCUser creates a new user account for an OIDC-authenticated user.
// The account is created without a password; the user can set one later via the
// password reset flow if they want to also sign in with email and password.
// Returns (nil, nil) if registration is administratively disabled.
func registerOIDCUser(ctx context.Context, email string) (*types.UserAccount, error) {
	if env.Registration() == env.RegistrationDisabled {
		return nil, nil
	}
	userAccount := types.UserAccount{
		Email:           email,
		EmailVerifiedAt: new(time.Now()),
	}
	if err := db.CreateUserAccountWithOrganization(ctx, &userAccount, &types.Organization{}); err != nil {
		return nil, fmt.Errorf("failed to create OIDC user: %w", err)
	}
	return &userAccount, nil
}

func verifyOIDCState(r *http.Request) (string, error) {
	state, err := uuid.Parse(r.URL.Query().Get("state"))
	if err != nil {
		return "", fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
	}
	pkceVerifier, createdAt, err := db.DeleteOIDCState(r.Context(), state)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return "", apierrors.ErrBadRequest
		}
		return "", err
	}
	if createdAt.Before(time.Now().UTC().Add(-1 * time.Minute)) {
		return "", fmt.Errorf("%w: got an OIDC state that is too old: %v, created_at: %v, now: %v",
			apierrors.ErrBadRequest, state, createdAt, time.Now().UTC())
	}
	return pkceVerifier, nil
}
