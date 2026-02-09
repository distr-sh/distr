package handlers

import (
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ContextRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireOrgAndRole).Get("/", getContextHandler)
}

func getContextHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	var orgs []types.OrganizationWithUserRole
	var err error

	// Super admins get all organizations
	if auth.IsSuperAdmin() {
		orgs, err = db.GetAllOrganizationsForSuperAdmin(ctx)
	} else {
		orgs, err = db.GetOrganizationsForUser(ctx, auth.CurrentUserID())
	}

	if err != nil {
		log.Error("failed to get organizations", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var joinDate time.Time
	var userRole *types.UserRole
	var customerOrgID *uuid.UUID

	if auth.IsSuperAdmin() {
		// Super admins: use current org's creation time as join date, no role
		joinDate = auth.CurrentOrg().CreatedAt
		userRole = util.PtrTo(types.UserRoleAdmin)
	} else {
		// Regular users: find their actual join date and role
		for _, org := range orgs {
			if org.ID == *auth.CurrentOrgID() {
				joinDate = org.JoinedOrgAt
				userRole = &org.UserRole
				customerOrgID = org.CustomerOrganizationID
				break
			}
		}
		if userRole == nil {
			userRole = auth.CurrentUserRole()
		}
	}

	var customerOrg *api.CustomerOrganization
	if customerOrgID != nil {
		if co, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID); err != nil {
			log.Error("failed to get customer organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else {
			mapped := mapping.CustomerOrganizationToAPI(co.CustomerOrganization)
			customerOrg = &mapped
		}
	}

	RespondJSON(w, api.ContextResponse{
		User: mapping.UserAccountToAPI(
			auth.CurrentUser().AsUserAccountWithRole(*userRole, customerOrgID, joinDate),
		),
		Organization:         mapping.OrganizationToAPI(*auth.CurrentOrg()),
		CustomerOrganization: customerOrg,
		AvailableContexts:    orgs,
	})
}
