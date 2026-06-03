package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
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
	email, emailVerified, err := oidcer.GetEmailForCode(ctx, provider, code, pkceVerifier, r)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC email extraction failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		user, err := db.GetUserAccountByEmail(ctx, email)
		if errors.Is(err, apierrors.ErrNotFound) {
			user, err = registerOIDCUser(ctx, email)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if user == nil {
			http.Redirect(w, r, redirectToLoginOIDCRegistrationDisabled, http.StatusFound)
			return nil
		}
		log = log.With(zap.Any("userId", user.ID))

		org, err := auth.EnsurePrimaryOrganization(ctx, *user)
		if err != nil {
			return err
		}

		if user.EmailVerifiedAt == nil && emailVerified {
			if err = db.UpdateUserAccountEmailVerified(ctx, user); err != nil {
				return err
			}
		}
		if _, tokenString, err := authjwt.GenerateDefaultToken(*user, org); err != nil {
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
		EmailVerifiedAt: util.PtrTo(time.Now()),
	}
	if _, err := db.CreateUserAccountWithOrganization(ctx, &userAccount); err != nil {
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
