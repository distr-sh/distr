package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func DeploymentTargetMetricsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Agents"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", getLatestDeploymentTargetMetrics).
		With(option.Description("Get latest deployment target metrics")).
		With(option.Response(http.StatusOK, []api.DeploymentTargetMetrics{}))
}

func getLatestDeploymentTargetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if deploymentTargetMetrics, err := db.GetLatestDeploymentTargetMetrics(
		ctx,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
	); err != nil {
		internalctx.GetLogger(ctx).Error("failed to get deployment target metrics", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(deploymentTargetMetrics, mapping.DeploymentTargetMetricsToAPI))
	}
}
