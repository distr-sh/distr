package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func SettingsRouter(r chiopenapi.Router) {
	r.Route("/user", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Settings"))

		r.Post("/", userSettingsUpdateHandler).
			With(option.Description("Update user settings")).
			With(option.Request(api.UpdateUserAccountRequest{})).
			With(option.Response(http.StatusOK, types.UserAccount{}))

		r.Post("/email", userSettingsUpdateEmailHandler()).
			With(option.Description("Update current user email address")).
			With(option.Request(api.UpdateUserAccountEmailRequest{})).
			With(option.Response(http.StatusAccepted, nil))
	})

	r.Route("/verify", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupHidden(true))

		r.With(requestVerificationMailRateLimitPerUser).
			Post("/request", userSettingsVerifyRequestHandler)

		r.Post("/confirm", userSettingsVerifyConfirmHandler)
	})

	r.Route("/mfa", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Security"))

		r.Post("/setup", mfaSetupHandler).
			With(option.Description("Setup a new TOTP secret for the current user. MFA must still be enabled afterwards")).
			With(option.Response(http.StatusOK, api.SetupMFAResponse{}))

		r.Post("/enable", mfaEnableHandler).
			With(option.Description("Enable MFA for the current user and receive recovery codes")).
			With(option.Request(api.EnableMFARequest{})).
			With(option.Response(http.StatusOK, api.EnableMFAResponse{}))

		r.Post("/disable", mfaDisableHandler).
			With(option.Description(
				"Disable MFA for the current user. This will also remove the TOTP secret and all recovery codes")).
			With(option.Request(api.DisableMFARequest{}))

		r.Post("/recovery-codes/regenerate", mfaRegenerateRecoveryCodesHandler).
			With(option.Description("Regenerate all recovery codes. This invalidates all existing codes")).
			With(option.Request(api.RegenerateMFARecoveryCodesRequest{})).
			With(option.Response(http.StatusOK, api.RegenerateMFARecoveryCodesResponse{}))

		r.Get("/recovery-codes/status", mfaRecoveryCodesStatusHandler).
			With(option.Description("Get the count of remaining unused recovery codes")).
			With(option.Response(http.StatusOK, api.MFARecoveryCodesStatusResponse{}))
	})

	r.Route("/tokens", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Access Tokens"))

		r.Use(middleware.RequireOrgAndRole)

		r.Get("/", getAccessTokensHandler()).
			With(option.Description("List all access tokens")).
			With(option.Response(http.StatusOK, []api.AccessToken{}))

		r.Post("/", createAccessTokenHandler()).
			With(option.Description("Create a new access token")).
			With(option.Request(api.CreateAccessTokenRequest{})).
			With(option.Response(http.StatusCreated, api.AccessTokenWithKey{}))

		r.Route("/{accessTokenId}", func(r chiopenapi.Router) {
			type AccessTokenIDRequest struct {
				AccessTokenID uuid.UUID `path:"accessTokenId"`
			}

			r.Delete("/", deleteAccessTokenHandler()).
				With(option.Description("Delete an access token")).
				With(option.Request(AccessTokenIDRequest{}))
		})
	})
}

func userSettingsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	body, err := JsonBody[api.UpdateUserAccountRequest](w, r)
	if err != nil {
		return
	}

	if err := body.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := auth.CurrentUser()
	isUpdateNeeded := false

	if body.Name != nil && *body.Name != user.Name {
		user.Name = *body.Name
		isUpdateNeeded = true
	}

	if body.Password != nil {
		user.Password = *body.Password
		if err := security.HashPassword(user); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to hash password", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		isUpdateNeeded = true
	}

	if body.ImageID != nil && !util.PtrEq(user.ImageID, body.ImageID) {
		user.ImageID = body.ImageID
		isUpdateNeeded = true
	}

	if user.EmailVerifiedAt == nil && auth.CurrentUserEmailVerified() {
		// because reset tokens can also verify the users email address
		user.EmailVerifiedAt = util.PtrTo(time.Now())
		isUpdateNeeded = true
	}

	if isUpdateNeeded {
		if err := db.UpdateUserAccount(ctx, user); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				log.Error("failed to update user", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	RespondJSON(w, user)
}

func userSettingsUpdateEmailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		mailer := internalctx.GetMailer(ctx)
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		user := auth.CurrentUser()

		body, err := JsonBody[api.UpdateUserAccountEmailRequest](w, r)
		if err != nil {
			return
		}

		if err := body.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if user.Email == body.Email {
			http.Error(w, "new email must be different from current email", http.StatusBadRequest)
			return
		}

		if exists, err := db.ExistsUserAccountWithEmail(ctx, body.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if exists {
			http.Error(w, "email already in use", http.StatusBadRequest)
			return
		}

		// Set new email on the UserAccount to generate a verification token
		// This is not saved to the DB yet!
		oldEmail := user.Email
		user.Email = body.Email
		_, token, err := authjwt.GenerateVerificationTokenValidFor(*user)
		if err != nil {
			log.Error("failed to send email verification", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "failed to generate verification token", http.StatusInternalServerError)
			return
		}
		user.Email = oldEmail

		branding, err := db.GetOrganizationBranding(ctx, *auth.CurrentOrgID())
		if err != nil && !errors.Is(err, apierrors.ErrNotFound) {
			log.Error("failed to get organization branding", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		owb := types.OrganizationWithBranding{Organization: *auth.CurrentOrg(), Branding: branding}

		msg := mail.New(
			mail.To(body.Email),
			mail.Subject("[Action required] Distr E-Mail address change"),
			mail.HtmlBodyTemplate(mailtemplates.UpdateEmail(*user, owb, token)),
		)

		if err := mailer.Send(ctx, msg); err != nil {
			log.Error("failed to send email verification", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func userSettingsVerifyRequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	userAccount := auth.CurrentUser()
	if userAccount.EmailVerifiedAt != nil {
		w.WriteHeader(http.StatusNoContent)
	} else if err := mailsending.SendUserVerificationMail(ctx, *userAccount, *auth.CurrentOrg()); err != nil {
		log.Error("failed to send verification mail", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		sentry.GetHubFromContext(ctx).CaptureException(err)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func userSettingsVerifyConfirmHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	userAccount := auth.CurrentUser()
	if !auth.CurrentUserEmailVerified() {
		http.Error(w, "token does not have verified claim", http.StatusForbidden)
		return
	}

	if userAccount.Email != auth.CurrentUserEmail() {
		userAccount.Email = auth.CurrentUserEmail()
	} else if userAccount.EmailVerifiedAt != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := db.UpdateUserAccountEmailVerified(ctx, userAccount); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "could not update user", http.StatusBadRequest)
		} else {
			log.Error("could not update user", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "could not update user", http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

var requestVerificationMailRateLimitPerUser = httprate.Limit(
	3,
	10*time.Minute,
	httprate.WithKeyFuncs(middleware.RateLimitUserIDKey),
)

var inviteUserRateLimiter = httprate.Limit(
	3,
	10*time.Minute,
	httprate.WithKeyFuncs(middleware.RateLimitUserIDKey, middleware.RateLimitPathValueKey("userId")),
)
