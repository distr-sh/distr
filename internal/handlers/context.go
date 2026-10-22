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
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ContextRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireOrgAndRole).Get("/", respond.Handler(getContextHandler))
	r.With(middleware.RequireOrgAndRole, middleware.RequireNonAccessToken).
		Get("/switch-targets", respond.Handler(getContextSwitchTargetsHandler))
}

func getContextHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	orgs, err := contextOrganizations(ctx, auth)
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

// getContextSwitchTargetsHandler lists, for the organizations the user may switch to, their logos and the
// customer and partner organizations whose lens the user may enter there. The memberships decide which, not the
// organization the session operates in right now, so the list stays the same while the session is in a lens.
func getContextSwitchTargetsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	orgs, err := contextOrganizations(ctx, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	orgIDs := make([]uuid.UUID, 0, len(orgs))
	ownTargets := make(map[uuid.UUID]lens.Target, len(orgs))
	var portalOrgIDs, partnerOrgIDs []uuid.UUID
	for _, org := range orgs {
		orgIDs = append(orgIDs, org.ID)
		ownTargets[org.ID] = lens.MembershipTarget(org)
		if auth.IsSuperAdmin() || org.CustomerOrganizationID == nil {
			portalOrgIDs = append(portalOrgIDs, org.ID)
			if org.HasFeature(types.FeaturePartnerManagement) {
				partnerOrgIDs = append(partnerOrgIDs, org.ID)
			}
		}
	}

	logoImageIDs, err := db.GetOrganizationLogoImageIDs(ctx, orgIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization logos: %w", err)
	}
	customerOrgs, err := db.GetCustomerOrganizationsByOrganizationIDs(ctx, portalOrgIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer organizations: %w", err)
	}
	partnerOrgs, err := db.GetPartnerOrganizationsByOrganizationIDs(ctx, partnerOrgIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get partner organizations: %w", err)
	}

	enterableCustomerOrgs := make(map[uuid.UUID][]types.CustomerOrganization)
	for _, co := range customerOrgs {
		if lens.MayEnterCustomer(ownTargets[co.OrganizationID], auth.IsSuperAdmin(), co.PartnerOrganizationID) {
			enterableCustomerOrgs[co.OrganizationID] = append(enterableCustomerOrgs[co.OrganizationID], co)
		}
	}
	enterablePartnerOrgs := make(map[uuid.UUID][]types.PartnerOrganization)
	for _, po := range partnerOrgs {
		if lens.MayEnterPartner(ownTargets[po.OrganizationID], auth.IsSuperAdmin()) {
			enterablePartnerOrgs[po.OrganizationID] = append(enterablePartnerOrgs[po.OrganizationID], po)
		}
	}

	response := api.ContextSwitchTargetsResponse{Organizations: []api.ContextSwitchOrganization{}}
	for _, org := range orgs {
		customers, partners := enterableCustomerOrgs[org.ID], enterablePartnerOrgs[org.ID]
		if logoImageIDs[org.ID] != nil || len(customers) > 0 || len(partners) > 0 {
			response.Organizations = append(response.Organizations,
				mapping.ContextSwitchOrganizationToAPI(org.ID, logoImageIDs[org.ID], customers, partners))
		}
	}
	return respond.JSON(response), nil
}

// contextOrganizations lists the organizations the user may switch to: every organization for a super admin and
// the memberships for everyone else, as far as the credential may reveal them.
func contextOrganizations(ctx context.Context, auth authinfo.AuthInfo) ([]types.OrganizationWithUserRole, error) {
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
