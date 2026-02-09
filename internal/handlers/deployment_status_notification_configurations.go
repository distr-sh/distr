package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func DeploymentStatusNotificationConfigurationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature)

	r.Get("/", getDeploymentStatusNotificationConfigurationsHandler()).
		With(option.Description("list all deployment status notification configurations")).
		With(option.Response(http.StatusOK, []types.DeploymentStatusNotificationConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Post("/", createDeploymentStatusNotificationConfigurationHandler()).
		With(option.Description("create a new deployment status notification configuration")).
		With(option.Request(types.DeploymentStatusNotificationConfiguration{})).
		With(option.Response(http.StatusOK, types.DeploymentStatusNotificationConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Route("/{id}", func(r chiopenapi.Router) {
			type IDRequest struct {
				ID string `path:"id"`
			}

			r.Put("/", updateDeploymentStatusNotificationHandler()).
				With(option.Description("update an existing deployment status notification configuration")).
				With(option.Request(struct {
					IDRequest
					types.DeploymentStatusNotificationConfiguration
				}{})).
				With(option.Response(http.StatusOK, types.DeploymentStatusNotificationConfiguration{}))

			r.Delete("/", deleteDeploymentStatusNotificationHandler()).
				With(option.Description("delete an existing deployment status notification configuration")).
				With(option.Request(IDRequest{}))
		})
}

func getDeploymentStatusNotificationConfigurationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		configs, err := db.GetDeploymentStatusNotificationConfigurations(
			ctx,
			*auth.CurrentOrgID(),
			auth.CurrentCustomerOrgID(),
		)
		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get notification configurations", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// TODO: implement api mapping
		RespondJSON(w, configs)
	}
}

func createDeploymentStatusNotificationConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		config, err := JsonBody[types.DeploymentStatusNotificationConfiguration](w, r)
		if err != nil {
			return
		}

		config.OrganizationID = *auth.CurrentOrgID()
		config.CustomerOrganizationID = auth.CurrentCustomerOrgID()

		if err := db.CreateDeploymentStatusNotificationConfiguration(ctx, &config); err != nil {
			internalctx.GetLogger(ctx).Error("failed to create notification configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// TODO: implement api mapping
		RespondJSON(w, config)
	}
}

func updateDeploymentStatusNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		config, err := JsonBody[types.DeploymentStatusNotificationConfiguration](w, r)
		if err != nil {
			return
		}

		config.ID = id
		config.OrganizationID = *auth.CurrentOrgID()
		config.CustomerOrganizationID = auth.CurrentCustomerOrgID()

		if err := db.UpdateDeploymentStatusNotificationConfiguration(ctx, &config); err != nil {
			internalctx.GetLogger(ctx).Error("failed to update notification configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// TODO: implement api mapping
		RespondJSON(w, config)
	}
}

func deleteDeploymentStatusNotificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := db.DeleteDeploymentStatusNotificationConfiguration(
			ctx,
			id,
			*auth.CurrentOrgID(),
			auth.CurrentCustomerOrgID(),
		); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			} else {
				internalctx.GetLogger(ctx).Error("failed to delete notification configuration", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
	}
}
