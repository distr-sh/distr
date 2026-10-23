package handlers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/controllermanifest"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/lens"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ContextRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireOrgAndRole).Get("/", respond.Handler(getContextHandler))
	r.With(middleware.RequireOrgAndRole, middleware.RequireNonAccessToken).
		Get("/lenses", respond.Handler(getLensesHandler))
}

func getContextHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	orgs, err := availableContexts(ctx, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	// Super admins have no role and no membership, so they are shown as admins who joined with the organization
	userRole := types.UserRoleAdmin
	joinDate := auth.CurrentOrg().CreatedAt
	if !auth.IsSuperAdmin() {
		userRole = *auth.CurrentUserRole()
		for _, org := range orgs {
			if org.ID == *auth.CurrentOrgID() {
				joinDate = org.JoinedOrgAt
				break
			}
		}
	}

	var customerOrg *api.CustomerOrganization
	var sidebarLinks []api.SidebarLink
	if co := cmp.Or(auth.LensCustomerOrg(), auth.CurrentCustomerOrg()); co != nil {
		customerOrg = new(mapping.CustomerOrganizationToAPI(*co))
		if links, err := db.GetSidebarLinks(ctx, co.ID); err != nil {
			log.Error("failed to get sidebar links", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else {
			sidebarLinks = mapping.List(links, mapping.SidebarLinkToAPI)
		}
	}

	var partnerOrg *api.PartnerOrganization
	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		if po, err := db.GetPartnerOrganizationByID(ctx, *partnerOrgID); err != nil {
			log.Error("failed to get partner organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else {
			partnerOrg = new(mapping.PartnerOrganizationToAPI(po.PartnerOrganization))
		}
	}

	orgID := *auth.CurrentOrgID()
	if billableUserCount, err := db.CountBillableUserAccountsByOrgID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to count billable user accounts: %w", err)
	} else if customerOrgCount, err := db.CountCustomerOrganizationsByOrganizationID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to count customer organizations: %w", err)
	} else {
		// The registry host is best-effort: without branding it still resolves custom domains
		// and falls back to the instance default.
		branding, err := db.GetOrganizationBranding(ctx, orgID)
		if err != nil && !errors.Is(err, distrerr.ErrNotFound) {
			log.Warn("failed to get organization branding", zap.Error(err))
		}
		registryHost := customdomains.RegistryDomainOrDefault(ctx, orgID, branding)
		controllerRepository := controllermanifest.DefaultImageRepository(ctx, orgID, branding)

		return respond.JSON(api.ContextResponse{
			User: mapping.UserAccountToAPI(
				auth.CurrentUser().AsUserAccountWithRole(
					userRole, auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(), joinDate,
				),
			),
			Organization: mapping.OrganizationToAPI(
				*auth.CurrentOrg(), billableUserCount, customerOrgCount, controllerRepository,
			),
			CustomerOrganization: customerOrg,
			PartnerOrganization:  partnerOrg,
			SidebarLinks:         sidebarLinks,
			AvailableContexts:    orgs,
			RegistryHost:         registryHost,
		}), nil
	}
}

// getLensesHandler lists the customer organizations of the current organization whose lens the user may enter. The
// own context decides which, not the lens the request is in right now, so the list stays the same while in a lens.
func getLensesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if ownOrg, resp, err := ownContext(ctx, auth, *auth.CurrentOrgID()); resp != nil || err != nil {
		return resp, err
	} else if customerOrgs, err := db.GetCustomerOrganizationsByOrganizationID(ctx, ownOrg.ID); err != nil {
		return nil, fmt.Errorf("failed to get customer organizations: %w", err)
	} else {
		own := lens.Own(*ownOrg)
		lenses := []api.CustomerLens{}
		for _, co := range customerOrgs {
			if lens.MayEnter(own, auth.IsSuperAdmin(), co.PartnerOrganizationID) {
				lenses = append(lenses, mapping.CustomerOrganizationToLens(co.CustomerOrganization))
			}
		}
		return respond.JSON(lenses), nil
	}
}

// availableContexts lists the organizations the user may switch to: every organization for a super admin and
// the memberships for everyone else, as far as the credential may reveal them.
func availableContexts(ctx context.Context, auth authinfo.AuthInfo) ([]types.OrganizationWithUserRole, error) {
	var orgs []types.OrganizationWithUserRole
	var err error
	if auth.IsSuperAdmin() {
		orgs, err = db.GetAllOrganizationsForSuperAdmin(ctx)
	} else {
		orgs, err = db.GetOrganizationsForUser(ctx, auth.CurrentUserID())
	}
	if err != nil {
		return nil, err
	}
	return visibleOrganizations(auth, orgs), nil
}
