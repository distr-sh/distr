package handlers

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ContextRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireOrgAndRole).Get("/", getContextHandler)
	r.With(middleware.RequireOrgAndRole, middleware.RequireNonAccessToken).
		Get("/switch-targets", getContextSwitchTargetsHandler)
}

func getContextHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	orgs, err := contextOrganizations(ctx, auth)
	if err != nil {
		log.Error("failed to get organizations", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
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
	if co := auth.CurrentCustomerOrg(); co != nil {
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
	billableUserCount, err := db.CountBillableUserAccountsByOrgID(ctx, orgID)
	if err != nil {
		log.Error("failed to count billable user accounts", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	customerOrgCount, err := db.CountCustomerOrganizationsByOrganizationID(ctx, orgID)
	if err != nil {
		log.Error("failed to count customer organizations", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// The registry host is best-effort: without branding it still resolves custom domains
	// and falls back to the instance default.
	branding, err := db.GetOrganizationBranding(ctx, orgID)
	if err != nil && !errors.Is(err, apierrors.ErrNotFound) {
		log.Warn("failed to get organization branding", zap.Error(err))
	}
	registryHost := customdomains.RegistryDomainOrDefault(ctx, orgID, branding)

	RespondJSON(w, api.ContextResponse{
		User: mapping.UserAccountToAPI(
			auth.CurrentUser().AsUserAccountWithRole(
				userRole, auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(), joinDate,
			),
		),
		Organization:         mapping.OrganizationToAPI(*auth.CurrentOrg(), billableUserCount, customerOrgCount),
		CustomerOrganization: customerOrg,
		PartnerOrganization:  partnerOrg,
		SidebarLinks:         sidebarLinks,
		AvailableContexts:    orgs,
		RegistryHost:         registryHost,
		CanCreateOrganization: mayCreateOrganization(
			auth, slices.ContainsFunc(orgs, isVendorMembership), env.Registration(),
		),
	})
}

// getContextSwitchTargetsHandler lists, for the organizations the user may switch to, their logos and the
// customer and partner portals the user may enter there. The memberships decide which portals, not the portal
// the session operates in right now, so the list stays the same while the session is in a foreign context.
func getContextSwitchTargetsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	orgs, err := contextOrganizations(ctx, auth)
	if err != nil {
		respondInternalError(w, r, err, "failed to get organizations")
		return
	}

	orgIDs := make([]uuid.UUID, 0, len(orgs))
	memberships := make(map[uuid.UUID]authinfo.Portal, len(orgs))
	var portalOrgIDs, partnerOrgIDs []uuid.UUID
	for _, org := range orgs {
		orgIDs = append(orgIDs, org.ID)
		memberships[org.ID] = authinfo.Portal{
			CustomerOrganizationID: org.CustomerOrganizationID,
			PartnerOrganizationID:  org.PartnerOrganizationID,
		}
		if auth.IsSuperAdmin() || org.CustomerOrganizationID == nil {
			portalOrgIDs = append(portalOrgIDs, org.ID)
			if org.HasFeature(types.FeaturePartnerManagement) {
				partnerOrgIDs = append(partnerOrgIDs, org.ID)
			}
		}
	}

	logoImageIDs, err := db.GetOrganizationLogoImageIDs(ctx, orgIDs)
	if err != nil {
		respondInternalError(w, r, err, "failed to get organization logos")
		return
	}
	customerOrgs, err := db.GetCustomerOrganizationsByOrganizationIDs(ctx, portalOrgIDs)
	if err != nil {
		respondInternalError(w, r, err, "failed to get customer organizations")
		return
	}
	partnerOrgs, err := db.GetPartnerOrganizationsByOrganizationIDs(ctx, partnerOrgIDs)
	if err != nil {
		respondInternalError(w, r, err, "failed to get partner organizations")
		return
	}

	enterableCustomerOrgs := make(map[uuid.UUID][]types.CustomerOrganization)
	for _, co := range customerOrgs {
		requested := authinfo.Portal{CustomerOrganizationID: &co.ID}
		if authinfo.MayEnter(memberships[co.OrganizationID], auth.IsSuperAdmin(), requested, co.PartnerOrganizationID) {
			enterableCustomerOrgs[co.OrganizationID] = append(enterableCustomerOrgs[co.OrganizationID], co)
		}
	}
	enterablePartnerOrgs := make(map[uuid.UUID][]types.PartnerOrganization)
	for _, po := range partnerOrgs {
		requested := authinfo.Portal{PartnerOrganizationID: &po.ID}
		if authinfo.MayEnter(memberships[po.OrganizationID], auth.IsSuperAdmin(), requested, nil) {
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
	RespondJSON(w, response)
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

func isVendorMembership(org types.OrganizationWithUserRole) bool {
	return org.CustomerOrganizationID == nil && org.PartnerOrganizationID == nil
}
