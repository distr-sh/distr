package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/blocklist"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/custommail"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/lens"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/turnstile"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/userauth"
	"github.com/distr-sh/distr/internal/util"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/getsentry/sentry-go"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func AuthRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(
		middleware.BlockIPs,
		httprate.LimitBy(
			10,
			1*time.Minute,
			httprate.JoinKeys(func(r *http.Request) (string, error) {
				return chimiddleware.GetClientIP(r.Context()), nil
			}, httprate.KeyByEndpoint),
		),
	)
	// The login methods available on a host are part of the host-resolved GET /api/public/v1/portal response.
	r.Post("/login", respond.Handler(authLoginHandler))
	r.Route("/oidc", AuthOIDCRouter)
	r.Post("/register", respond.Handler(authRegisterHandler))
	r.Post("/reset", respond.Handler(authResetPasswordHandler))
	r.Group(func(r chiopenapi.Router) {
		r.Use(
			auth.Authentication.Middleware,
			middleware.SetSentryUserFromUserAuth,
			middleware.RequireEmailVerified,
			middleware.RequireOrgAndRole,
			middleware.RequireNonAccessToken,
		)
		r.Post("/switch-context", respond.Handler(authSwitchContextHandler))
		r.Post("/enter-lens", respond.Handler(authEnterLensHandler))
	})
	r.Group(func(r chiopenapi.Router) {
		r.Use(auth.Authentication.Middleware, middleware.SetSentryUserFromUserAuth)

		// Accepting an invitation and confirming a password reset must not be behind RequireEmailVerified:
		// the user's DB record may not be verified yet at this point. Both handlers set the password, verify
		// the email when the token carries a verified claim, and return a regular login token so the frontend
		// can log the user in directly. RequireTokenScope pins each endpoint to its dedicated special token,
		// so an org-scoped login token or a PAT cannot be used to change an account's password.
		r.With(middleware.RequireTokenScope(authjwt.TokenScopeInvite)).
			Post("/invite/accept", respond.Handler(authAcceptInviteHandler))
		r.With(middleware.RequireTokenScope(authjwt.TokenScopePasswordReset)).
			Post("/reset/confirm", respond.Handler(authResetConfirmHandler))

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
			).Post("/request", respond.Handler(authVerifyRequestHandler))
			r.Post("/confirm", respond.Handler(authVerifyConfirmHandler))
		})

		r.Get("/status", respond.Handler(authStatusHandler)).With(option.Hidden(true))
	})
}

func authStatusHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	return respond.JSON(map[string]any{"active": auth.CurrentUser().Activated}), nil
}

func authVerifyRequestHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	userAccount := auth.CurrentUser()
	if userAccount.EmailVerifiedAt != nil {
		return respond.NoContent(), nil
	} else if err := mailsending.SendUserVerificationMail(
		ctx, *userAccount, *auth.CurrentOrg(), auth.CurrentCustomerOrgID(), true); err != nil {
		return nil, fmt.Errorf("failed to send verification mail: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func authVerifyConfirmHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authn := auth.Authentication.Require(ctx)
	if !authn.CurrentUserEmailVerified() {
		return respond.Forbidden("token does not have verified claim"), nil
	} else if blocklist.EmailBlocked(authn.CurrentUserEmail()) {
		return respond.Forbidden(blocklist.EmailBlockedMessage), nil
	} else if err := userauth.VerifyUserEmail(ctx, authn.CurrentUser(), authn.CurrentUserEmail()); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return respond.BadRequest("could not update user"), nil
		} else {
			return nil, fmt.Errorf("could not update user: %w", err)
		}
	} else {
		return respond.NoContent(), nil
	}
}

func authAcceptInviteHandler(r *http.Request) (respond.Responder, error) {
	if body, resp := decodeJSON[api.AuthAcceptInviteRequest](r); resp != nil {
		return resp, nil
	} else if err := body.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		return setPasswordAndLogin(r, userauth.SetInitialUserPassword, body.Password, body.Name, body.MFACode)
	}
}

func authResetConfirmHandler(r *http.Request) (respond.Responder, error) {
	if body, resp := decodeJSON[api.AuthResetPasswordConfirmRequest](r); resp != nil {
		return resp, nil
	} else if err := body.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		return setPasswordAndLogin(r, userauth.SetUserPassword, body.Password, nil, body.MFACode)
	}
}

// setPasswordAndLogin sets (and persists) the given password and optional name for the authenticated user,
// verifies their email when the credential carries a verified email claim, and responds with a fresh login
// token so the frontend can log the user in directly. It is shared by the invite-accept and reset-confirm flows.
// An account with MFA enabled has to pass the same check as on a regular login before any of this happens,
// since a reset or invitation link only proves control over the mailbox.
func setPasswordAndLogin(
	r *http.Request,
	setPassword func(ctx context.Context, user *types.UserAccount, password string, name *string) error,
	password string,
	name, mfaCode *string,
) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authn := auth.Authentication.Require(ctx)
	user := authn.CurrentUser()

	var token string
	if blocklist.EmailBlocked(user.Email) {
		return respond.Forbidden(blocklist.EmailBlockedMessage), nil
	} else if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := userauth.VerifyMFA(ctx, *user, mfaCode); err != nil {
			return err
		}
		if err := setPassword(ctx, user, password, name); err != nil {
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
	}); err != nil {
		if errors.Is(err, userauth.ErrMFARequired) {
			return respond.JSON(api.AuthLoginResponse{RequiresMFA: true}), nil
		} else if errors.Is(err, userauth.ErrMFACodeInvalid) {
			// A 401 would make the frontend discard the invite or reset token and send the user back to the
			// "link expired" page, so a wrong code must not be answered with one.
			return respond.BadRequest("invalid MFA code or recovery code"), nil
		} else if errors.Is(err, distrerr.ErrNotFound) {
			return respond.BadRequest("could not update user"), nil
		} else if errors.Is(err, distrerr.ErrConflict) {
			return respond.BadRequest("this invitation has already been accepted, please log in instead"), nil
		} else if errors.Is(err, subscription.ErrGlobalOrganizationLimitReached) {
			log.Warn("could not set password, global organization limit reached")
			return respond.BadRequest(subscription.GlobalOrganizationLimitReachedMessage), nil
		} else {
			return nil, fmt.Errorf("failed to set password: %w", err)
		}
	} else {
		// When the email could not be verified from the token (e.g. an invitation link shared manually instead of
		// delivered via email) and verification is required, send the verification mail so the user receives it
		// without having to request it manually after being redirected to the verification page.
		if user.EmailVerifiedAt == nil && env.UserEmailVerificationRequired() {
			if org, err := userauth.PrimaryOrganization(ctx, *user); err != nil {
				log.Warn("could not resolve organization for verification mail", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
			} else if err := mailsending.SendUserVerificationMail(
				ctx, *user, org.Organization, org.CustomerOrganizationID, true); err != nil {
				log.Warn("could not send verification mail", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		}

		return respond.JSON(api.AuthLoginResponse{Token: token}), nil
	}
}

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

// authEnterLensHandler switches the session to the lens of a customer organization of the organization it
// operates in.
func authEnterLensHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	request, resp := decodeJSON[api.AuthEnterLensRequest](r)
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	}

	requested := request.CustomerOrganizationID
	if auth.IsLens() && util.PtrEq(&requested, auth.CurrentCustomerOrgID()) {
		return respond.NoContent(), nil
	} else if ownOrg, resp, err := ownContext(ctx, auth, *auth.CurrentOrgID()); resp != nil || err != nil {
		return resp, err
	} else if own := lens.Own(*ownOrg); lens.IsOwn(own, &requested) {
		return respond.BadRequest("this is your own portal, return to it instead"), nil
	} else if _, err := lens.Enter(ctx, ownOrg.ID, own, auth.IsSuperAdmin(), requested); errors.Is(
		err, lens.ErrForbidden,
	) {
		return respond.Forbidden(err.Error()), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to enter lens: %w", err)
	} else {
		lensOrg := *ownOrg
		lensOrg.CustomerOrganizationID = &requested
		lensOrg.PartnerOrganizationID = nil
		if tokenString, err := generateContextToken(auth, lensOrg, true); err != nil {
			return nil, err
		} else {
			return respond.JSON(api.AuthLoginResponse{Token: tokenString}), nil
		}
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

var invalidCredentials = respond.BadRequest("invalid username or password")

func authLoginHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	request, resp := decodeJSON[api.AuthLoginRequest](r)
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if blocklist.EmailBlocked(request.Email) {
		return respond.Forbidden(blocklist.EmailBlockedMessage), nil
	}

	var response api.AuthLoginResponse
	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if user, err := db.GetUserAccountByEmail(ctx, request.Email); errors.Is(err, distrerr.ErrNotFound) {
			return invalidCredentials, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to get user account: %w", err)
		} else if err := security.VerifyPassword(*user, request.Password); err != nil {
			return invalidCredentials, nil
		} else if err := userauth.VerifyMFA(ctx, *user, request.MFACode); errors.Is(err, userauth.ErrMFARequired) {
			return respond.JSON(api.AuthLoginResponse{RequiresMFA: true}), nil
		} else if errors.Is(err, userauth.ErrMFACodeInvalid) {
			return respond.Unauthorized("invalid MFA code or recovery code"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to verify MFA: %w", err)
		} else if token, err := userauth.GenerateLoginToken(ctx, *user); errors.Is(
			err, subscription.ErrGlobalOrganizationLimitReached,
		) {
			log.Warn("user login rejected, global organization limit reached", zap.Any("userId", user.ID))
			return respond.BadRequest(subscription.GlobalOrganizationLimitReachedMessage), nil
		} else if err != nil {
			return nil, fmt.Errorf("token creation failed: %w", err)
		} else if err := db.UpdateUserAccountLastLoggedIn(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("failed to update last login: %w", err)
		} else {
			response = api.AuthLoginResponse{Token: token, RedirectURL: loginAppDomainRedirect(ctx, r, *user, token)}
			return nil, nil
		}
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(response), nil
	}
}

func authRegisterHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	if env.Registration() == env.RegistrationDisabled {
		return respond.Forbidden("registration is disabled"), nil
	}

	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		return nil, fmt.Errorf("could not resolve host for registration: %w", err)
	} else if !host.registrationAllowed() {
		return respond.Forbidden("registration is not available on this domain"), nil
	}

	request, resp := decodeJSON[api.AuthRegistrationRequest](r)
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if blocklist.EmailBlocked(request.Email) {
		return respond.Forbidden(blocklist.EmailBlockedMessage), nil
	} else if resp := verifyRegistrationChallenge(ctx, host, request.TurnstileToken); resp != nil {
		return resp, nil
	}

	userAccount := types.UserAccount{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	}
	org := types.Organization{
		Name: request.OrganizationName,
	}
	var token string

	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if reached, err := subscription.IsGlobalOrganizationLimitReached(ctx); err != nil {
			return nil, fmt.Errorf("could not check global organization limit: %w", err)
		} else if reached {
			return respond.BadRequest(subscription.GlobalOrganizationLimitReachedMessage), nil
		} else if err := security.HashPassword(&userAccount); err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		} else if err := db.CreateUserAccountWithOrganization(ctx, &userAccount, &org); errors.Is(
			err, distrerr.ErrAlreadyExists,
		) {
			return respond.BadRequest("an account with this email address already exists, please log in instead"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to create user account: %w", err)
		} else if token, err = userauth.GenerateLoginToken(ctx, userAccount); err != nil {
			return nil, fmt.Errorf("failed to generate login token: %w", err)
		} else {
			return nil, nil
		}
	}); resp != nil || err != nil {
		return resp, err
	}

	// When email verification is required the user is redirected to the verification page after logging in,
	// so they need the verification mail. When it is disabled they are logged in directly and the mail would
	// be pointless.
	if env.UserEmailVerificationRequired() {
		if err := mailsending.SendUserVerificationMail(ctx, userAccount, org, nil, false); err != nil {
			log.Warn("could not send verification mail", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
	}

	return respond.JSON(api.AuthLoginResponse{Token: token}), nil
}

func verifyRegistrationChallenge(ctx context.Context, host portalHost, token string) respond.Responder {
	if host.turnstileSiteKey() == nil {
		return nil
	} else if err := turnstile.Verify(ctx, token, chimiddleware.GetClientIP(ctx)); err != nil {
		internalctx.GetLogger(ctx).Info("turnstile verification failed", zap.Error(err))
		// The frontend shows the message of a 4xx response as-is, so it has to be one a user can act on.
		return respond.Forbidden("could not verify that you are human, please reload the page and try again")
	} else {
		return nil
	}
}

func authResetPasswordHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	if request, resp := decodeJSON[api.AuthResetPasswordRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if blocklist.EmailBlocked(request.Email) {
		return respond.Forbidden(blocklist.EmailBlockedMessage), nil
	} else if user, err := db.GetUserAccountByEmail(ctx, request.Email); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			log.Info("password reset for non-existing user", zap.String("email", request.Email))
			return respond.NoContent(), nil
		} else {
			return nil, fmt.Errorf("could not send reset mail: %w", err)
		}
	} else if orgs, err := db.GetOrganizationsForUser(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("could not send reset mail: %w", err)
	} else if _, token, err := authjwt.GenerateResetToken(*user); err != nil {
		return nil, fmt.Errorf("could not send reset mail: %w", err)
	} else {
		var organization *types.OrganizationWithBranding
		var customerOrgID *uuid.UUID
		mailer := internalctx.GetMailer(ctx)
		mailOpts := []mailx.MailOpt{
			mailx.To(user.Email),
			mailx.Subject("Password reset"),
		}
		if len(orgs) > 0 {
			customerOrgID = orgs[0].CustomerOrganizationID
			if result, err := db.GetOrganizationWithBranding(ctx, orgs[0].ID); err != nil {
				return nil, fmt.Errorf("failed to get org with branding: %w", err)
			} else {
				organization = result
			}

			// The mail is sent through the organization of the user's first membership: the reset
			// request is unauthenticated, so there is no current organization to resolve.
			if result, err := custommail.MailerForOrganization(ctx, organization.ID); err != nil {
				return nil, fmt.Errorf("could not send reset mail: %w", err)
			} else {
				mailer = result
			}

			if from, err := custommail.FromAddressOrDefault(ctx, organization.ID, organization.Branding); err == nil {
				mailOpts = append(mailOpts, mailx.From(*from))
			} else {
				log.Warn("error parsing custom from address", zap.Error(err))
			}
		}
		mailOpts = append(mailOpts,
			mailx.HtmlBodyTemplate(mailtemplates.PasswordReset(ctx, *user, organization, customerOrgID, token)))
		if err := mailer.Send(ctx, mailOpts...); err != nil {
			return nil, fmt.Errorf("could not send reset mail: %w", err)
		} else {
			return respond.NoContent(), nil
		}
	}
}
