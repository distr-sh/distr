package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/blocklist"
	"github.com/distr-sh/distr/internal/custommail"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func SettingsRouter(r chiopenapi.Router) {
	r.Route("/user", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Settings"))

		r.Post("/", respond.Handler(userSettingsUpdateHandler)).
			With(option.Description("Update user settings")).
			With(option.Request(api.UpdateUserAccountRequest{})).
			With(option.Response(http.StatusOK, types.UserAccount{}))

		r.With(middleware.RequireNonAccessToken, middleware.RequireNonOidcToken).
			Post("/email", respond.Handler(userSettingsUpdateEmailHandler)).
			With(option.Description("Update current user email address")).
			With(option.Request(api.UpdateUserAccountEmailRequest{})).
			With(option.Response(http.StatusAccepted, nil))

		r.Route("/oidc-identities", func(r chiopenapi.Router) {
			type OIDCIdentityIDRequest struct {
				OIDCIdentityID uuid.UUID `path:"oidcIdentityId"`
			}

			r.Get("/", respond.Handler(getOIDCIdentitiesHandler)).
				With(option.Description("List the identity provider accounts connected to the current user")).
				With(option.Response(http.StatusOK, []api.UserAccountOIDCIdentity{}))

			r.With(middleware.RequireNonAccessToken, middleware.RequireNonOidcToken).
				Delete("/{oidcIdentityId}", respond.Handler(deleteOIDCIdentityHandler)).
				With(option.Description("Disconnect an identity provider account from the current user")).
				With(option.Request(OIDCIdentityIDRequest{}))
		})
	})

	r.Route("/mfa", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Security"))

		// Enrollment is gated because it needs no password and would hand whoever performs it a second
		// factor the account's owner does not have. Disabling MFA and regenerating the recovery codes
		// verify the password, which is proof of ownership on its own.
		r.With(middleware.RequireNonAccessToken, middleware.RequireNonOidcToken).
			Post("/setup", respond.Handler(mfaSetupHandler)).
			With(option.Description("Setup a new TOTP secret for the current user. MFA must still be enabled afterwards")).
			With(option.Response(http.StatusOK, api.SetupMFAResponse{}))

		r.With(middleware.RequireNonAccessToken, middleware.RequireNonOidcToken).
			Post("/enable", respond.Handler(mfaEnableHandler)).
			With(option.Description("Enable MFA for the current user and receive recovery codes")).
			With(option.Request(api.EnableMFARequest{})).
			With(option.Response(http.StatusOK, api.EnableMFAResponse{}))

		r.Post("/disable", respond.Handler(mfaDisableHandler)).
			With(option.Description(
				"Disable MFA for the current user. This will also remove the TOTP secret and all recovery codes")).
			With(option.Request(api.DisableMFARequest{}))

		r.Post("/recovery-codes/regenerate", respond.Handler(mfaRegenerateRecoveryCodesHandler)).
			With(option.Description("Regenerate all recovery codes. This invalidates all existing codes")).
			With(option.Request(api.RegenerateMFARecoveryCodesRequest{})).
			With(option.Response(http.StatusOK, api.RegenerateMFARecoveryCodesResponse{}))

		r.Get("/recovery-codes/status", respond.Handler(mfaRecoveryCodesStatusHandler)).
			With(option.Description("Get the count of remaining unused recovery codes")).
			With(option.Response(http.StatusOK, api.MFARecoveryCodesStatusResponse{}))
	})

	r.Route("/tokens", func(r chiopenapi.Router) {
		// A token is created in the web UI by the person it belongs to, and a client that already
		// holds one has no business minting another, so these endpoints are not part of the API.
		r.WithOptions(option.GroupHidden())

		r.Use(middleware.RequireOrgAndRole, middleware.RequireNonAccessToken)

		r.Get("/", respond.Handler(getAccessTokensHandler))

		r.With(middleware.BlockSuperAdmin).Post("/", respond.Handler(createAccessTokenHandler))

		r.Route("/{accessTokenId}", func(r chiopenapi.Router) {
			r.With(middleware.BlockSuperAdmin).Patch("/", respond.Handler(patchAccessTokenHandler))

			r.With(middleware.BlockSuperAdmin).Delete("/", respond.Handler(deleteAccessTokenHandler))

			r.With(middleware.BlockSuperAdmin).Delete("/key", respond.Handler(deleteAccessTokenKeyHandler))

			r.Route("/secrets", func(r chiopenapi.Router) {
				r.With(middleware.BlockSuperAdmin).Post("/", respond.Handler(createAccessTokenSecretHandler))

				r.With(middleware.BlockSuperAdmin).Delete("/{slot}", respond.Handler(deleteAccessTokenSecretHandler))
			})
		})
	})
}

func userSettingsUpdateHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if body, resp := decodeJSON[api.UpdateUserAccountRequest](r); resp != nil {
		return resp, nil
	} else if err := body.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if body.Password != nil && auth.IsAccessToken() {
		// Only the password is a credential change; the name and the image stay available to every session.
		return respond.Forbidden(middleware.AccessTokenBlockedMessage), nil
	} else if body.Password != nil && auth.IsCustomOIDCSession() {
		return respond.Forbidden(middleware.OidcSessionBlockedMessage), nil
	} else {
		user := auth.CurrentUser()
		isUpdateNeeded := false

		if body.Name != nil && *body.Name != user.Name {
			user.Name = *body.Name
			isUpdateNeeded = true
		}

		if body.Password != nil {
			user.Password = *body.Password
			if err := security.HashPassword(user); err != nil {
				return nil, fmt.Errorf("failed to hash password: %w", err)
			}
			isUpdateNeeded = true
		}

		if body.ImageID != nil && !util.PtrEq(user.ImageID, body.ImageID) {
			user.ImageID = body.ImageID
			isUpdateNeeded = true
		}

		if isUpdateNeeded {
			if err := db.UpdateUserAccount(ctx, user); err != nil {
				if errors.Is(err, distrerr.ErrNotFound) {
					return respond.BadRequest(err.Error()), nil
				} else {
					return nil, fmt.Errorf("failed to update user: %w", err)
				}
			}
		}

		return respond.JSON(user), nil
	}
}

func userSettingsUpdateEmailHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	user := auth.CurrentUser()

	if body, resp := decodeJSON[api.UpdateUserAccountEmailRequest](r); resp != nil {
		return resp, nil
	} else if err := body.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if blocklist.EmailBlocked(body.Email) {
		return respond.Forbidden(blocklist.EmailBlockedMessage), nil
	} else if user.Email == body.Email {
		return respond.BadRequest("new email must be different from current email"), nil
	} else if exists, err := db.ExistsUserAccountWithEmail(ctx, body.Email); err != nil {
		return nil, fmt.Errorf("failed to check whether email is in use: %w", err)
	} else if exists {
		return respond.BadRequest("email already in use"), nil
	} else {
		// Set new email on the UserAccount to generate a verification token
		// This is not saved to the DB yet!
		oldEmail := user.Email
		user.Email = body.Email
		_, token, err := authjwt.GenerateVerificationTokenValidFor(*user)
		if err != nil {
			return nil, fmt.Errorf("failed to generate verification token: %w", err)
		}
		user.Email = oldEmail

		if mailer, err := custommail.MailerForOrganization(ctx, *auth.CurrentOrgID()); err != nil {
			return nil, fmt.Errorf("failed to resolve mailer for email change: %w", err)
		} else if err := mailer.Send(ctx,
			mailx.To(body.Email),
			mailx.Subject("[Action required] Distr E-Mail address change"),
			mailx.HtmlBodyTemplate(mailtemplates.UpdateEmail(
				ctx, *user, *auth.CurrentOrgWithBranding(), auth.CurrentCustomerOrgID(), token,
			)),
		); err != nil {
			return nil, fmt.Errorf("failed to send email verification: %w", err)
		} else {
			return respond.Status(http.StatusAccepted), nil
		}
	}
}
