package handlers

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func OrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Organizations"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", respond.Handler(getOrganizations)).
		With(option.Description("List all organizations for current user")).
		With(option.Response(http.StatusOK, []types.OrganizationWithUserRole{}))
}

func getOrganizations(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if orgs, err := db.GetOrganizationsForUser(ctx, auth.CurrentUserID()); err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	} else {
		return respond.JSON(visibleOrganizations(auth, orgs)), nil
	}
}

// visibleOrganizations drops the organizations a credential must not reveal: one that is confined to the
// organization it was issued for (see authinfo.AuthInfo.OrganizationScoped) may not learn which other
// organizations the account is a member of, since it must not act in them either.
func visibleOrganizations(
	auth authinfo.AuthInfo,
	orgs []types.OrganizationWithUserRole,
) []types.OrganizationWithUserRole {
	if !auth.OrganizationScoped() {
		return orgs
	}
	return slices.DeleteFunc(orgs, func(org types.OrganizationWithUserRole) bool {
		return auth.CurrentOrgID() == nil || org.ID != *auth.CurrentOrgID()
	})
}
