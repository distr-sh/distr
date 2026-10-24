package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/lens"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func LensesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireOrgAndRole, middleware.RequireNonAccessToken).
		Get("/", respond.Handler(getLensesHandler))
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
