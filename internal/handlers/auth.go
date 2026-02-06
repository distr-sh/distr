package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

func AuthRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(httprate.Limit(
		10,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
	))
	r.Route("/login", func(r chiopenapi.Router) {
		r.Post("/", authLoginHandler)
		r.Get("/config", authLoginConfigHandler())
	})
	r.Route("/oidc", AuthOIDCRouter)
	r.Post("/register", authRegisterHandler)
	r.Post("/reset", authResetPasswordHandler)
	r.With(middleware.SentryUser, auth.Authentication.Middleware, middleware.RequireOrgAndRole).
		Post("/switch-context", authSwitchContextHandler())
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
			Organization:           *org,
			UserRole:               user.UserRole,
			CustomerOrganizationID: user.CustomerOrganizationID,
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

		var orgs []types.OrganizationWithUserRole
		if user.IsSuperAdmin {
			orgs, err = db.GetAllOrganizationsForSuperAdmin(ctx)
		} else {
			orgs, err = db.GetOrganizationsForUser(ctx, user.ID)
		}

		if err != nil {
			return err
		}

		var org types.OrganizationWithUserRole
		if len(orgs) == 0 {
			if !user.IsSuperAdmin {
				org.Name = user.Email
				org.UserRole = types.UserRoleAdmin
				if err := db.CreateOrganization(ctx, &org.Organization); err != nil {
					return err
				} else if err := db.CreateUserAccountOrganizationAssignment(
					ctx, user.ID, org.ID, org.UserRole, org.CustomerOrganizationID); err != nil {
					return err
				}
			} else {
				return errors.New("super admin has no organizations, this should never happen")
			}
		} else {
			org = orgs[0]
			if user.LastUsedOrganizationID != nil {
				for _, o := range orgs {
					if o.ID == *user.LastUsedOrganizationID {
						org = o
						break
					}
				}
			}
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

		if _, tokenString, err := authjwt.GenerateDefaultToken(*user, org); err != nil {
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
		var org *types.Organization

		if err := db.RunTx(ctx, func(ctx context.Context) error {
			if err := security.HashPassword(&userAccount); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				w.WriteHeader(http.StatusInternalServerError)
				return err
			} else if org, err = db.CreateUserAccountWithOrganization(ctx, &userAccount); err != nil {
				if errors.Is(err, apierrors.ErrAlreadyExists) {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					sentry.GetHubFromContext(ctx).CaptureException(err)
					w.WriteHeader(http.StatusInternalServerError)
				}
				return err
			}
			return nil
		}); err != nil {
			log.Warn("user registration failed", zap.Error(err))
			return
		}

		if err := mailsending.SendUserVerificationMail(ctx, userAccount, *org); err != nil {
			log.Warn("could not send verification mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}

		w.WriteHeader(http.StatusNoContent)
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
		mailOpts := []mail.MailOpt{
			mail.To(user.Email),
			mail.Subject("Password reset"),
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

			if from, err := customdomains.EmailFromAddressParsedOrDefault(organization.Organization); err == nil {
				mailOpts = append(mailOpts, mail.From(*from))
			} else {
				log.Warn("error parsing custom from address", zap.Error(err))
			}
		}
		mailOpts = append(mailOpts, mail.HtmlBodyTemplate(mailtemplates.PasswordReset(*user, organization, token)))
		if err := mailer.Send(ctx, mail.New(mailOpts...)); err != nil {
			log.Warn("could not send reset mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
