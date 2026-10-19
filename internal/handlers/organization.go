package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/blocklist"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func OrganizationRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Organizations"))
	r.Use(middleware.RequireOrgAndRole)

	r.Get("/", respond.Handler(getOrganization)).
		With(option.Description("Get current organization")).
		With(option.Response(http.StatusOK, api.OrganizationResponse{}))

	r.With(middleware.BlockCrossOrganizationAction).
		Post("/", respond.Handler(createOrganization)).
		With(option.Description("Create a new organization")).
		With(option.Request(api.CreateOrganizationRequest{})).
		With(option.Response(http.StatusOK, api.CreateOrganizationResponse{}))

	r.With(middleware.BlockSuperAdmin, middleware.RequireVendor, middleware.RequireAdmin).
		Put("/", respond.Handler(updateOrganization)).
		With(option.Description("Update current organization")).
		With(option.Request(api.CreateUpdateOrganizationRequest{})).
		With(option.Response(http.StatusOK, types.Organization{}))

	r.With(
		middleware.RequireVendor,
		middleware.RequireAdmin,
		middleware.BlockCrossOrganizationAction,
		middleware.BlockSuperAdminUnlessOrganizationExpired,
	).
		Delete("/", respond.Handler(deleteOrganizationHandler)).
		With(option.Description("Delete current organization"))

	r.Route("/branding", OrganizationBrandingRouter)

	r.Route("/webhook", func(r chiopenapi.Router) {
		r.Use(middleware.RequireVendor, middleware.VendorBillingFeatureMiddleware)

		r.Get("/", respond.Handler(getOrganizationWebhook)).
			With(option.Description("Get vendor webhook configuration status")).
			With(option.Response(http.StatusOK, api.OrganizationWebhookResponse{}))

		r.With(middleware.RequireAdmin, middleware.BlockSuperAdmin).
			Put("/", respond.Handler(updateOrganizationWebhook)).
			With(option.Description("Set vendor Stripe webhook secret")).
			With(option.Request(api.UpdateOrganizationWebhookRequest{}))
	})
}

func getOrganization(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	if billableUserCount, err := db.CountBillableUserAccountsByOrgID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to count billable user accounts: %w", err)
	} else if customerOrgCount, err := db.CountCustomerOrganizationsByOrganizationID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to count customer organizations: %w", err)
	} else {
		return respond.JSON(mapping.OrganizationToAPI(*auth.CurrentOrg(), billableUserCount, customerOrgCount)), nil
	}
}

// mayCreateOrganization keeps vendors from creating organizations while registration is hidden or disabled, since
// another organization is what a sign-up would get them. Super admins may create one regardless.
func mayCreateOrganization(
	auth authinfo.AuthInfo,
	hasVendorMembership bool,
	registration env.RegistrationMode,
) bool {
	return !auth.OrganizationScoped() &&
		(auth.IsSuperAdmin() || (hasVendorMembership && registration == env.RegistrationEnabled))
}

var slugNotAvailable = respond.BadRequest("Slug is not available")

func updateOrganization(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	var org *types.Organization
	if body, resp := decodeJSON[api.CreateUpdateOrganizationRequest](r); resp != nil {
		return resp, nil
	} else if resp := validateOrganizationRequest(body); resp != nil {
		return resp, nil
	} else if resp, err := runTxRR(ctx, func(ctx context.Context) (respond.Responder, error) {
		var err error
		if org, err = db.GetOrganizationByID(ctx, *auth.CurrentOrgID()); err != nil {
			return nil, fmt.Errorf("failed to get organization: %w", err)
		} else if org.Slug != nil && *org.Slug != "" && (body.Slug == nil || *body.Slug == "") {
			return respond.BadRequest("slug can not get unset"), nil
		} else if err := applyOrganizationUpdate(ctx, org, body); errors.Is(err, distrerr.ErrConflict) {
			return slugNotAvailable, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to update organization: %w", err)
		} else {
			return nil, nil
		}
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(org), nil
	}
}

func createOrganization(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if hasVendorMembership, err := db.ExistsVendorOrganizationWithUserID(ctx, auth.CurrentUserID()); err != nil {
		return nil, fmt.Errorf("failed to check if user is vendor: %w", err)
	} else if !mayCreateOrganization(auth, hasVendorMembership, env.Registration()) {
		return respond.Forbidden("you are not allowed to create organizations"), nil
	} else if body, resp := decodeJSON[api.CreateOrganizationRequest](r); resp != nil {
		return resp, nil
	} else if resp := validateOrganizationRequest(body.CreateUpdateOrganizationRequest); resp != nil {
		return resp, nil
	} else if resp := validateOrganizationAdmin(auth, body.AdminEmail); resp != nil {
		return resp, nil
	} else if reached, err := subscription.IsGlobalOrganizationLimitReached(ctx); err != nil {
		return nil, fmt.Errorf("could not check global organization limit: %w", err)
	} else if reached {
		return respond.BadRequest(subscription.GlobalOrganizationLimitReachedMessage), nil
	} else {
		organization := mapping.CreateOrganizationRequestToInternal(body)
		var inviteURL string

		if err := db.RunTx(ctx, func(ctx context.Context) error {
			if err := db.CreateOrganization(ctx, &organization); err != nil {
				return err
			} else if !auth.IsSuperAdmin() {
				return db.CreateUserAccountOrganizationAssignment(
					ctx, auth.CurrentUserID(), organization.ID, types.UserRoleAdmin, nil, nil)
			} else if orgWithBranding, err := db.GetOrganizationWithBranding(ctx, organization.ID); err != nil {
				return err
			} else {
				// Super admins hand the organization over instead of joining it, since their sessions stay read-only.
				_, inviteURL, err = inviteUserAccount(
					ctx, *orgWithBranding, body.AdminEmail, "", types.UserRoleAdmin, nil, nil)
				return err
			}
		}); errors.Is(err, distrerr.ErrConflict) {
			return slugNotAvailable, nil
		} else if err != nil {
			return nil, fmt.Errorf("could not create organization: %w", err)
		} else {
			return respond.JSON(mapping.OrganizationToCreateResponse(organization, inviteURL)), nil
		}
	}
}

func validateOrganizationAdmin(auth authinfo.AuthInfo, adminEmail string) respond.Responder {
	if !auth.IsSuperAdmin() {
		if adminEmail != "" {
			return respond.BadRequest("only super admins can choose the admin of a new organization")
		}
	} else if adminEmail == "" {
		return respond.BadRequest("the email address of the organization's admin is required")
	} else if err := validation.ValidateEmail(adminEmail); err != nil {
		return respond.BadRequest(err.Error())
	} else if blocklist.EmailBlocked(adminEmail) {
		return respond.Forbidden(blocklist.EmailBlockedMessage)
	}
	return nil
}

func validateOrganizationRequest(organization api.CreateUpdateOrganizationRequest) respond.Responder {
	if organization.Name == "" {
		return respond.BadRequest("name is required")
	}
	if organization.Slug != nil {
		if err := validation.ValidateSlug(*organization.Slug); err != nil {
			return respond.BadRequest(err.Error())
		}
	}
	return nil
}

func applyOrganizationUpdate(
	ctx context.Context,
	org *types.Organization,
	request api.CreateUpdateOrganizationRequest,
) error {
	needsUpdate := false

	if org.Name != request.Name {
		org.Name = request.Name
		needsUpdate = true
	}

	if !util.PtrEq(org.Slug, request.Slug) {
		org.Slug = request.Slug
		needsUpdate = true
	}

	if !util.PtrEq(org.DockerPreConnectScript, request.DockerPreConnectScript) {
		org.DockerPreConnectScript = request.DockerPreConnectScript
		needsUpdate = true
	}

	if !util.PtrEq(org.DockerPostConnectScript, request.DockerPostConnectScript) {
		org.DockerPostConnectScript = request.DockerPostConnectScript
		needsUpdate = true
	}

	if request.DockerConnectScriptIsSudo != org.DockerConnectScriptIsSudo {
		org.DockerConnectScriptIsSudo = request.DockerConnectScriptIsSudo
		needsUpdate = true
	}

	if !util.PtrEq(org.KubernetesPreConnectScript, request.KubernetesPreConnectScript) {
		org.KubernetesPreConnectScript = request.KubernetesPreConnectScript
		needsUpdate = true
	}

	if !util.PtrEq(org.KubernetesPostConnectScript, request.KubernetesPostConnectScript) {
		org.KubernetesPostConnectScript = request.KubernetesPostConnectScript
		needsUpdate = true
	}

	if request.ArtifactVersionMutable != org.HasFeature(types.FeatureArtifactVersionMutable) {
		org.SetFeature(types.FeatureArtifactVersionMutable, request.ArtifactVersionMutable)
		needsUpdate = true
	}

	if request.PrePostScriptsEnabled != org.HasFeature(types.FeaturePrePostScripts) {
		org.SetFeature(types.FeaturePrePostScripts, request.PrePostScriptsEnabled)
		needsUpdate = true
	}

	if needsUpdate {
		return db.UpdateOrganization(ctx, org)
	}

	return nil
}

func deleteOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if err := db.SetOrganizationDeletedAtNow(ctx, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("could not soft-delete organization: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func getOrganizationWebhook(r *http.Request) (respond.Responder, error) {
	authCtx := auth.Authentication.Require(r.Context())
	return respond.JSON(api.OrganizationWebhookResponse{
		Configured: authCtx.CurrentOrg().StripeWebhookSecretConfigured,
	}), nil
}

func updateOrganizationWebhook(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	if body, resp := decodeJSON[api.UpdateOrganizationWebhookRequest](r); resp != nil {
		return resp, nil
	} else if err := body.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		secret := dbcrypto.StringPtr(body.WebhookSecret)
		if err := db.SetOrganizationStripeWebhookSecret(ctx, *authCtx.CurrentOrgID(), secret); err != nil {
			return nil, fmt.Errorf("failed to update stripe webhook secret: %w", err)
		} else {
			return respond.NoContent(), nil
		}
	}
}
