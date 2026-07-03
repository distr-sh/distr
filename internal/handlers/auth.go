package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/userauth"
	"github.com/getsentry/sentry-go"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

func AuthRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(httprate.LimitBy(
		10,
		1*time.Minute,
		httprate.JoinKeys(func(r *http.Request) (string, error) {
			return chimiddleware.GetClientIP(r.Context()), nil
		}, httprate.KeyByEndpoint),
	))
	r.Route("/login", func(r chiopenapi.Router) {
		r.Post("/", authLoginHandler)
		r.Get("/config", authLoginConfigHandler())
	})
	r.Route("/oidc", AuthOIDCRouter)
	r.Post("/register", authRegisterHandler)
	r.Post("/reset", authResetPasswordHandler)
	r.With(
		auth.Authentication.Middleware,
		middleware.SetSentryUserFromUserAuth,
		middleware.RequireEmailVerified,
		middleware.RequireOrgAndRole,
	).Post("/switch-context", authSwitchContextHandler())
	r.Group(func(r chiopenapi.Router) {
		r.Use(auth.Authentication.Middleware, middleware.SetSentryUserFromUserAuth)

		// Accepting an invitation and confirming a password reset must not be behind RequireEmailVerified:
		// the user's DB record may not be verified yet at this point. Both handlers set the password, verify
		// the email when the token carries a verified claim, and return a regular login token so the frontend
		// can log the user in directly. RequireTokenScope pins each endpoint to its dedicated special token,
		// so an org-scoped login token or a PAT cannot be used to change an account's password.
		r.With(middleware.RequireTokenScope(authjwt.TokenScopeInvite)).
			Post("/invite/accept", authAcceptInviteHandler)
		r.With(middleware.RequireTokenScope(authjwt.TokenScopePasswordReset)).
			Post("/reset/confirm", authResetConfirmHandler)

		r.Route("/verify", func(r chiopenapi.Router) {
			requestVerificationMailRateLimitPerUser := httprate.LimitBy(
				3,
				10*time.Minute,
				middleware.RateLimitUserIDKey,
			)
			r.With(
				requestVerificationMailRateLimitPerUser,
				middleware.BlockSuperAdmin,
				middleware.RequireOrgAndRole,
			).Post("/request", authVerifyRequestHandler)
			r.Post("/confirm", authVerifyConfirmHandler)
		})

		r.Get("/status", authStatusHandler).With(option.Hidden(true))
	})
}

func authStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	userAccount := auth.CurrentUser()
	RespondJSON(w, map[string]any{
		"active": userAccount.PasswordHash != nil,
	})
}

func authVerifyRequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	userAccount := auth.CurrentUser()
	if userAccount.EmailVerifiedAt != nil {
		w.WriteHeader(http.StatusNoContent)
	} else if err := mailsending.SendUserVerificationMail(ctx, *userAccount, *auth.CurrentOrg(), true); err != nil {
		log.Error("failed to send verification mail", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		sentry.GetHubFromContext(ctx).CaptureException(err)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func authVerifyConfirmHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authn := auth.Authentication.Require(ctx)
	if !authn.CurrentUserEmailVerified() {
		http.Error(w, "token does not have verified claim", http.StatusForbidden)
		return
	}

	if err := userauth.VerifyUserEmail(ctx, authn.CurrentUser(), authn.CurrentUserEmail()); err != nil {
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

func authAcceptInviteHandler(w http.ResponseWriter, r *http.Request) {
	body, err := JsonBody[api.AuthAcceptInviteRequest](w, r)
	if err != nil {
		return
	}
	if err := body.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setPasswordAndLogin(w, r, body.Password, body.Name)
}

func authResetConfirmHandler(w http.ResponseWriter, r *http.Request) {
	body, err := JsonBody[api.AuthResetPasswordConfirmRequest](w, r)
	if err != nil {
		return
	}
	if err := body.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setPasswordAndLogin(w, r, body.Password, nil)
}

// setPasswordAndLogin sets (and persists) the given password and optional name for the authenticated user,
// verifies their email when the credential carries a verified email claim, and responds with a fresh login
// token so the frontend can log the user in directly. It is shared by the invite-accept and reset-confirm flows.
func setPasswordAndLogin(w http.ResponseWriter, r *http.Request, password string, name *string) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authn := auth.Authentication.Require(ctx)
	user := authn.CurrentUser()

	var token string
	err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := userauth.SetUserPassword(ctx, user, password, name); err != nil {
			return err
		}
		if authn.CurrentUserEmailVerified() {
			if err := userauth.VerifyUserEmail(ctx, user, authn.CurrentUserEmail()); err != nil {
				return err
			}
		}
		if err := db.UpdateUserAccountLastLoggedIn(ctx, user.ID); err != nil {
			return err
		}
		var err error
		token, err = userauth.GenerateLoginToken(ctx, *user)
		return err
	})
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "could not update user", http.StatusBadRequest)
		} else {
			log.Error("failed to set password", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	// When the email could not be verified from the token (e.g. an invitation link shared manually instead of
	// delivered via email) and verification is required, send the verification mail so the user receives it
	// without having to request it manually after being redirected to the verification page.
	if user.EmailVerifiedAt == nil && env.UserEmailVerificationRequired() {
		if org, err := userauth.PrimaryOrganization(ctx, *user); err != nil {
			log.Warn("could not resolve organization for verification mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else if err := mailsending.SendUserVerificationMail(ctx, *user, org.Organization, true); err != nil {
			log.Warn("could not send verification mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
	}

	RespondJSON(w, api.AuthLoginResponse{Token: token})
}

func authSwitchContextHandler() func(writer http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		request, err := JsonBody[api.AuthSwitchContextRequest](w, r)
		if err != nil {
			return
		} else if request.OrganizationID == uuid.Nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		auth := auth.Authentication.Require(ctx)
		if *auth.CurrentOrgID() == request.OrganizationID {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Super admins can switch to any organization
		if auth.IsSuperAdmin() {
			user, err := db.GetUserAccountByID(ctx, auth.CurrentUserID())
			if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				log.Error("failed to get user account", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			org, err := db.GetOrganizationByID(ctx, request.OrganizationID)
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, "organization not found", http.StatusNotFound)
				return
			} else if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				log.Error("failed to get organization", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			_, tokenString, err := authjwt.GenerateDefaultToken(*user, types.OrganizationWithUserRole{
				Organization:           *org,
				UserRole:               types.UserRole(""), // Super admins don't have a role
				CustomerOrganizationID: nil,
			})
			if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				log.Error("failed to generate token", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if err := db.UpdateUserAccountLastUsedOrganizationID(ctx, user.ID, request.OrganizationID); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				log.Error("failed to update last used organization ID", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			RespondJSON(w, api.AuthLoginResponse{Token: tokenString})
			return
		}

		// Regular users: validate membership
		if user, org, err := db.GetUserAccountAndOrg(
			ctx, auth.CurrentUserID(), request.OrganizationID); errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		} else if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("context switch failed", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else if _, tokenString, err := authjwt.GenerateDefaultToken(user.AsUserAccount(), types.OrganizationWithUserRole{
			Organization:           org.Organization,
			UserRole:               user.UserRole,
			CustomerOrganizationID: user.CustomerOrganizationID,
			PartnerOrganizationID:  user.PartnerOrganizationID,
		}); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to generate token", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else if err := db.UpdateUserAccountLastUsedOrganizationID(ctx, user.ID, request.OrganizationID); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to update last used organization ID", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else {
			RespondJSON(w, api.AuthLoginResponse{Token: tokenString})
		}
	}
}

func authLoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	request, err := JsonBody[api.AuthLoginRequest](w, r)
	if err != nil {
		return
	}
	err = db.RunTx(ctx, func(ctx context.Context) error {
		user, err := db.GetUserAccountByEmail(ctx, request.Email)
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "invalid username or password", http.StatusBadRequest)
			return nil
		} else if err != nil {
			return err
		}
		log = log.With(zap.Any("userId", user.ID))
		if err = security.VerifyPassword(*user, request.Password); err != nil {
			http.Error(w, "invalid username or password", http.StatusBadRequest)
			return nil
		}

		if user.MFAEnabled {
			if request.MFACode == nil {
				RespondJSON(w, api.AuthLoginResponse{RequiresMFA: true})
				return nil
			}

			if user.MFASecret == nil {
				// this can never happen because we guard against it with a db constraint
				sentry.GetHubFromContext(ctx).CaptureException(errors.New("user has mfa enabled but no secret"))
				http.Error(w, "MFA configuration error", http.StatusInternalServerError)
				return nil
			}

			valid := totp.Validate(*request.MFACode, *user.MFASecret)

			if !valid {
				normalized := security.NormalizeRecoveryCode(*request.MFACode)
				codes, err := db.GetUnusedMFARecoveryCodes(ctx, user.ID)
				if err != nil {
					return fmt.Errorf("failed to get recovery codes: %w", err)
				}

				var matchedCodeID *uuid.UUID
				for _, code := range codes {
					if security.VerifyRecoveryCode(normalized, code.CodeSalt, code.CodeHash) {
						matchedCodeID = &code.ID
						break
					}
				}

				if matchedCodeID == nil {
					http.Error(w, "invalid MFA code or recovery code", http.StatusUnauthorized)
					return nil
				}

				if err := db.MarkMFARecoveryCodeAsUsed(ctx, *matchedCodeID); err != nil {
					return err
				}
			}
		}

		if tokenString, err := userauth.GenerateLoginToken(ctx, *user); err != nil {
			return fmt.Errorf("token creation failed: %w", err)
		} else if err = db.UpdateUserAccountLastLoggedIn(ctx, user.ID); err != nil {
			return err
		} else {
			RespondJSON(w, api.AuthLoginResponse{Token: tokenString})
			return nil
		}
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Warn("user login failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func authLoginConfigHandler() http.HandlerFunc {
	resp := struct {
		RegistrationEnabled  bool `json:"registrationEnabled"`
		OIDCGithubEnabled    bool `json:"oidcGithubEnabled"`
		OIDCGoogleEnabled    bool `json:"oidcGoogleEnabled"`
		OIDCMicrosoftEnabled bool `json:"oidcMicrosoftEnabled"`
		OIDCGenericEnabled   bool `json:"oidcGenericEnabled"`
	}{
		RegistrationEnabled:  env.Registration() == env.RegistrationEnabled,
		OIDCGithubEnabled:    env.OIDCGithubEnabled(),
		OIDCGoogleEnabled:    env.OIDCGoogleEnabled(),
		OIDCMicrosoftEnabled: env.OIDCMicrosoftEnabled(),
		OIDCGenericEnabled:   env.OIDCGenericEnabled(),
	}
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, resp)
	}
}

func authRegisterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	if env.Registration() == env.RegistrationDisabled {
		http.Error(w, "registration is disabled", http.StatusForbidden)
		return
	}

	if request, err := JsonBody[api.AuthRegistrationRequest](w, r); err != nil {
		return
	} else if err := request.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		userAccount := types.UserAccount{
			Name:     request.Name,
			Email:    request.Email,
			Password: request.Password,
		}
		org := types.Organization{
			Name: strings.TrimSpace(request.OrganizationName),
		}
		var token string

		if err := db.RunTx(ctx, func(ctx context.Context) error {
			if err := security.HashPassword(&userAccount); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				w.WriteHeader(http.StatusInternalServerError)
				return err
			} else if err = db.CreateUserAccountWithOrganization(ctx, &userAccount, &org); err != nil {
				if errors.Is(err, apierrors.ErrAlreadyExists) {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					sentry.GetHubFromContext(ctx).CaptureException(err)
					w.WriteHeader(http.StatusInternalServerError)
				}
				return err
			} else if token, err = userauth.GenerateLoginToken(ctx, userAccount); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				w.WriteHeader(http.StatusInternalServerError)
				return err
			}
			return nil
		}); err != nil {
			log.Warn("user registration failed", zap.Error(err))
			return
		}

		// When email verification is required the user is redirected to the verification page after logging in,
		// so they need the verification mail. When it is disabled they are logged in directly and the mail would
		// be pointless.
		if env.UserEmailVerificationRequired() {
			if err := mailsending.SendUserVerificationMail(ctx, userAccount, org, false); err != nil {
				log.Warn("could not send verification mail", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		}

		RespondJSON(w, api.AuthLoginResponse{Token: token})
	}
}

func authResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	mailer := internalctx.GetMailer(ctx)
	if request, err := JsonBody[api.AuthResetPasswordRequest](w, r); err != nil {
		return
	} else if err := request.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if user, err := db.GetUserAccountByEmail(ctx, request.Email); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			log.Info("password reset for non-existing user", zap.String("email", request.Email))
			w.WriteHeader(http.StatusNoContent)
		} else {
			log.Warn("could not send reset mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
		}
	} else if orgs, err := db.GetOrganizationsForUser(ctx, user.ID); err != nil {
		log.Error("could not send reset mail", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	} else if _, token, err := authjwt.GenerateResetToken(*user); err != nil {
		log.Error("could not send reset mail", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	} else {
		var organization *types.OrganizationWithBranding
		mailOpts := []mailx.MailOpt{
			mailx.To(user.Email),
			mailx.Subject("Password reset"),
		}
		if len(orgs) > 0 {
			if result, err := db.GetOrganizationWithBranding(ctx, orgs[0].ID); err != nil {
				err = fmt.Errorf("failed to get org with branding: %w", err)
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			} else {
				organization = result
			}

			if from, err := customdomains.EmailFromAddressParsedOrDefault(organization.Branding); err == nil {
				mailOpts = append(mailOpts, mailx.From(*from))
			} else {
				log.Warn("error parsing custom from address", zap.Error(err))
			}
		}
		mailOpts = append(mailOpts, mailx.HtmlBodyTemplate(mailtemplates.PasswordReset(ctx, *user, organization, token)))
		if err := mailer.Send(ctx, mailOpts...); err != nil {
			log.Warn("could not send reset mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
