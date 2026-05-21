package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

const redirectToLoginOIDCFailed = "/login?reason=oidc-failed"

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
			http.Redirect(w, r, "/register?reason=oidc-user-not-found&email="+email, http.StatusFound)
			return nil
		} else if err != nil {
			return err
		}
		log = log.With(zap.Any("userId", user.ID))

		var org types.OrganizationWithUserRole
		orgs, err := db.GetOrganizationsForUser(ctx, user.ID)
		if err != nil {
			return err
		} else if len(orgs) < 1 {
			// TODO deduplicate (regular login)
			org.Name = user.Email
			org.UserRole = types.UserRoleAdmin
			if err := db.CreateOrganization(ctx, &org.Organization); err != nil {
				return err
			} else if err := db.CreateUserAccountOrganizationAssignment(
				ctx, user.ID, org.ID, org.UserRole, org.CustomerOrganizationID); err != nil {
				return err
			}
		} else {
			org = orgs[0]
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

func verifyOIDCState(r *http.Request) (string, error) {
	state, err := uuid.Parse(r.URL.Query().Get("state"))
	if err != nil {
		return "", err
	}
	pkceVerifier, createdAt, err := db.DeleteOIDCState(r.Context(), state)
	if err != nil {
		return "", err
	}
	if createdAt.Before(time.Now().UTC().Add(-1 * time.Minute)) {
		return "", fmt.Errorf("got an OIDC state that is too old: %v, created_at: %v, now: %v",
			state, createdAt, time.Now().UTC())
	}
	return pkceVerifier, nil
}
