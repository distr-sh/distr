package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func PartnerOrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Partners"))
	r.With(middleware.RequireVendor, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getPartnerOrganizationsHandler()).
			With(option.Description("List all partner organizations")).
			With(option.Response(http.StatusOK, []api.PartnerOrganizationWithUsage{}))

		r.Route("/{partnerOrganizationId}", func(r chiopenapi.Router) {
			type PartnerOrganizationIDRequest struct {
				PartnerOrganizationID uuid.UUID `path:"partnerOrganizationId"`
			}

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
				r.Put("/", updatePartnerOrganizationHandler()).
					With(option.Description("Update a partner organization")).
					With(option.Request(struct {
						PartnerOrganizationIDRequest
						api.CreateUpdatePartnerOrganizationRequest
					}{})).
					With(option.Response(http.StatusOK, api.PartnerOrganization{}))
				r.Delete("/", deletePartnerOrganizationHandler()).
					With(option.Description("Delete a partner organization")).
					With(option.Request(PartnerOrganizationIDRequest{}))
			})
		})

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createPartnerOrganizationHandler()).
			With(option.Description("Create a new partner organization")).
			With(option.Request(api.CreateUpdatePartnerOrganizationRequest{})).
			With(option.Response(http.StatusOK, api.PartnerOrganization{}))
	})
}

func getPartnerOrganizationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		partnerOrgs, err := db.GetPartnerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
		if err != nil {
			log.Error("failed to get partner orgs", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		RespondJSON(w, mapping.List(partnerOrgs, mapping.PartnerOrganizationWithUsageToAPI))
	}
}

func createPartnerOrganizationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		request, err := JsonBody[api.CreateUpdatePartnerOrganizationRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		partnerOrg := types.PartnerOrganization{
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
		}
		if err := db.CreatePartnerOrganization(ctx, &partnerOrg); err != nil {
			log.Error("failed to create partner org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		RespondJSON(w, mapping.PartnerOrganizationToAPI(partnerOrg))
	}
}

func updatePartnerOrganizationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("partnerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		request, err := JsonBody[api.CreateUpdatePartnerOrganizationRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		partnerOrg := types.PartnerOrganization{
			ID:             id,
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
		}
		if err := db.UpdatePartnerOrganization(ctx, &partnerOrg); errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Error("failed to update partner org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.PartnerOrganizationToAPI(partnerOrg))
		}
	}
}

//nolint:dupl
func deletePartnerOrganizationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("partnerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		if err := db.DeletePartnerOrganizationWithID(ctx, id, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "partner organization is not empty", http.StatusConflict)
		} else if err != nil {
			log.Error("failed to delete partner org", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
