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

func AlertConfigurationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature)

	r.Get("/", getAlertConfigurationsHandler()).
		With(option.Description("list all alert configurations")).
		With(option.Response(http.StatusOK, []types.AlertConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Post("/", createAlertConfigurationHandler()).
		With(option.Description("create a new alert configuration")).
		With(option.Request(types.AlertConfiguration{})).
		With(option.Response(http.StatusOK, types.AlertConfiguration{}))

	r.Route("/{id}", func(r chiopenapi.Router) {
		type IDRequest struct {
			ID string `path:"id"`
		}

		r.Put("/", updateAlertConfigurationHandler()).
			With(option.Description("update an existing alert configuration")).
			With(option.Request(struct {
				IDRequest
				types.AlertConfiguration
			}{})).
			With(option.Response(http.StatusOK, types.AlertConfiguration{}))

		r.Delete("/", deleteAlertConfigurationHandler()).
			With(option.Description("delete an existing alert configuration")).
			With(option.Request(IDRequest{}))
	})
}

func getAlertConfigurationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		configs, err := db.GetAlertConfigurations(
			ctx,
			*auth.CurrentOrgID(),
			auth.CurrentCustomerOrgID(),
		)
		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get alert configurations", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// TODO: implement api mapping
		RespondJSON(w, configs)
	}
}

func createAlertConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		config, err := JsonBody[types.AlertConfiguration](w, r)
		if err != nil {
			return
		}

		config.OrganizationID = *auth.CurrentOrgID()
		config.CustomerOrganizationID = auth.CurrentCustomerOrgID()

		if err := db.CreateAlertConfiguration(ctx, &config); err != nil {
			internalctx.GetLogger(ctx).Error("failed to create alert configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// TODO: implement api mapping
		RespondJSON(w, config)
	}
}

func updateAlertConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		config, err := JsonBody[types.AlertConfiguration](w, r)
		if err != nil {
			return
		}

		config.ID = id
		config.OrganizationID = *auth.CurrentOrgID()
		config.CustomerOrganizationID = auth.CurrentCustomerOrgID()

		if err := db.UpdateAlertConfiguration(ctx, &config); err != nil {
			internalctx.GetLogger(ctx).Error("failed to update alert configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// TODO: implement api mapping
		RespondJSON(w, config)
	}
}

func deleteAlertConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := db.DeleteAlertConfiguration(
			ctx,
			id,
			*auth.CurrentOrgID(),
			auth.CurrentCustomerOrgID(),
		); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			} else {
				internalctx.GetLogger(ctx).Error("failed to delete alert configuration", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
	}
}
