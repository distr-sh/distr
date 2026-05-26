package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func OrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Organizations"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", getOrganizations).
		With(option.Description("List all organizations for current user")).
		With(option.Response(http.StatusOK, []types.OrganizationWithRole{}))
}

func getOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if orgs, err := db.GetOrganizationsForUser(ctx, auth.CurrentUserID()); err != nil {
		internalctx.GetLogger(ctx).Error("failed to get organizations", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(orgs, mapping.OrganizationWithRoleToAPI))
	}
}
