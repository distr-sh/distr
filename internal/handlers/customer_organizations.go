package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func CustomerOrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Customers"))
	r.With(middleware.RequireVendorOrPartner, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getCustomerOrganizationsHandler()).
			With(option.Description("List all customer organizations")).
			With(option.Response(http.StatusOK, []api.CustomerOrganizationWithUsage{}))

		r.Route("/{customerOrganizationId}", func(r chiopenapi.Router) {
			type CustomerOrganizationIDRequest struct {
				CustomerOrganizationID uuid.UUID `path:"customerOrganizationId"`
			}

			r.Route("/links", SidebarLinksRouter)

			r.With(middleware.RequireVendorOrPartner, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Put("/", updateCustomerOrganizationHandler()).
				With(option.Description("Update a customer organization")).
				With(option.Request(struct {
					CustomerOrganizationIDRequest
					api.CreateUpdateCustomerOrganizationRequest
				}{})).
				With(option.Response(http.StatusOK, api.CustomerOrganization{}))

			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin,
				middleware.BlockSuperAdmin, middleware.PartnerManagementFeatureMiddleware).Group(func(r chiopenapi.Router) {
				r.Put("/partner", assignCustomerToPartnerHandler()).
					With(option.Description("Assign or unassign a partner organization for a customer organization")).
					With(option.Request(struct {
						CustomerOrganizationIDRequest
						api.AssignCustomerToPartnerRequest
					}{})).
					With(option.Response(http.StatusOK, api.CustomerOrganization{}))
				r.Delete("/", deleteCustomerOrganizationHandler()).
					With(option.Description("Delete a customer organization")).
					With(option.Request(CustomerOrganizationIDRequest{}))
			})
		})

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createCustomerOrganizationHandler()).
			With(option.Description("Create a new customer organization")).
			With(option.Request(api.CreateUpdateCustomerOrganizationRequest{})).
			With(option.Response(http.StatusOK, api.CustomerOrganization{}))
	})
}

func getCustomerOrganizationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		var customerOrganizations []types.CustomerOrganizationWithUsage
		var err error
		if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
			customerOrganizations, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
		} else {
			customerOrganizations, err = db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
		}

		if err != nil {
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
			OrganizationID:        *auth.CurrentOrgID(),
			Name:                  request.Name,
			ImageID:               request.ImageID,
			PartnerOrganizationID: auth.CurrentPartnerOrgID(),
		}

		err = db.RunTx(ctx, func(ctx context.Context) error {
			if limitReached, err := subscription.IsCustomerOrganizationLimitReached(ctx, *auth.CurrentOrg()); err != nil {
				return err
			} else if limitReached {
				return apierrors.NewForbidden("customer limit reached")
			}
			return db.CreateCustomerOrganization(ctx, &customerOrganization)
		})

		if err == nil {
			RespondJSON(w, mapping.CustomerOrganizationToAPI(customerOrganization))
		} else if errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if errors.Is(err, apierrors.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			log.Error("failed to create customer org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

		if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
			co, coErr := db.GetCustomerOrganizationByID(ctx, id)
			if coErr != nil || !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
				http.Error(w, "customer is not assigned to your partner organization", http.StatusForbidden)
				return
			}
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

func assignCustomerToPartnerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		request, err := JsonBody[api.AssignCustomerToPartnerRequest](w, r)
		if err != nil {
			return
		}

		if request.PartnerOrganizationID != nil {
			if err := db.ValidatePartnerOrgBelongsToOrg(ctx, *request.PartnerOrganizationID, *auth.CurrentOrgID()); err != nil {
				if errors.Is(err, db.ErrPartnerOrgNotInOrg) {
					http.Error(w, "partner organization not found", http.StatusBadRequest)
				} else {
					log.Error("failed to check partner in org", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				return
			}
		}

		err = db.SetCustomerOrganizationPartner(ctx, id, *auth.CurrentOrgID(), request.PartnerOrganizationID)
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Error("failed to assign partner to customer org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			customerOrg, err := db.GetCustomerOrganizationByID(ctx, id)
			if err != nil {
				log.Error("failed to get customer org after partner assignment", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			RespondJSON(w, mapping.CustomerOrganizationToAPI(customerOrg.CustomerOrganization))
		}
	}
}

//nolint:dupl
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
