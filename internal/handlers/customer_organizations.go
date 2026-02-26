package handlers

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func CustomerOrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Customers"))
	r.With(middleware.RequireVendor, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getCustomerOrganizationsHandler()).
			With(option.Description("List all customer organizations")).
			With(option.Response(http.StatusOK, []api.CustomerOrganizationWithUsage{}))

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Post("/", createCustomerOrganizationHandler()).
				With(option.Description("Create a new customer organization")).
				With(option.Request(api.CreateUpdateCustomerOrganizationRequest{})).
				With(option.Response(http.StatusOK, api.CustomerOrganization{}))
			r.Route("/{customerOrganizationId}", func(r chiopenapi.Router) {
				type CustomerOrganizationIDRequest struct {
					CustomerOrganizationID uuid.UUID `path:"customerOrganizationId"`
				}

				r.Put("/", updateCustomerOrganizationHandler()).
					With(option.Description("Update a customer organization")).
					With(option.Request(struct {
						CustomerOrganizationIDRequest
						api.CreateUpdateCustomerOrganizationRequest
					}{})).
					With(option.Response(http.StatusOK, api.CustomerOrganization{}))
				r.Delete("/", deleteCustomerOrganizationHandler()).
					With(option.Description("Delete a customer organization")).
					With(option.Request(CustomerOrganizationIDRequest{}))
			})
		})
	})
}

func getCustomerOrganizationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		if customerOrganizations, err := db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID()); err != nil {
			log.Error("failed to get customer orgs", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.List(customerOrganizations, mapping.CustomerOrganizationWithUsageToAPI))
		}
	}
}

func createCustomerOrganizationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		request, err := JsonBody[api.CreateUpdateCustomerOrganizationRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		customerOrganization := types.CustomerOrganization{
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
			ImageID:        request.ImageID,
		}

		err = db.RunTx(ctx, func(ctx context.Context) error {
			if limitReached, err := subscription.IsCustomerOrganizationLimitReached(ctx, *auth.CurrentOrg()); err != nil {
				log.Error("failed to get customer orgs", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			} else if limitReached {
				err = errors.New("customer limit reached")
				http.Error(w, err.Error(), http.StatusForbidden)
				return err
			}

			if err := db.CreateCustomerOrganization(ctx, &customerOrganization); err != nil {
				log.Error("failed to create customer org", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.CustomerOrganizationToAPI(customerOrganization))
		}
	}
}

func updateCustomerOrganizationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		request, err := JsonBody[api.CreateUpdateCustomerOrganizationRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var features []types.CustomerOrganizationFeature
		if request.Features == nil {
			features = []types.CustomerOrganizationFeature{
				types.CustomerOrganizationFeatureDeploymentTargets,
				types.CustomerOrganizationFeatureArtifacts,
			}
		} else {
			features = request.Features
		}

		if slices.Contains(features, types.CustomerOrganizationFeatureSupport) {
			exists, err := db.ExistsSupportBundleConfiguration(ctx, *auth.CurrentOrgID())
			if err != nil {
				log.Error("failed to check support bundle configuration", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !exists {
				http.Error(w, "support bundle configuration must be set up before enabling support feature", http.StatusBadRequest)
				return
			}
		}

		customerOrganization := types.CustomerOrganization{
			ID:             id,
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
			ImageID:        request.ImageID,
			Features:       features,
		}

		if err := db.UpdateCustomerOrganization(ctx, &customerOrganization); err != nil {
			log.Error("failed to update customer org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.CustomerOrganizationToAPI(customerOrganization))
		}
	}
}

func deleteCustomerOrganizationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		if err := db.DeleteCustomerOrganizationWithID(ctx, id, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "customer is not empty", http.StatusConflict)
		} else if err != nil {
			log.Error("failed to delete customer org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
