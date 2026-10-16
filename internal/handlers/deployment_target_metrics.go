package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func DeploymentTargetMetricsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Controllers"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", respond.Handler(getLatestDeploymentTargetMetrics)).
		With(option.Description("Get latest deployment target metrics")).
		With(option.Response(http.StatusOK, []api.DeploymentTargetMetrics{}))
}

func getLatestDeploymentTargetMetrics(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if deploymentTargetMetrics, err := db.GetLatestDeploymentTargetMetrics(
		ctx,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
		auth.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment target metrics: %w", err)
	} else {
		return respond.JSON(mapping.List(deploymentTargetMetrics, mapping.DeploymentTargetMetricsToAPI)), nil
	}
}
