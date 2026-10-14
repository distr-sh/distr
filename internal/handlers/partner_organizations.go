package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func PartnerOrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Partners"))
	r.With(middleware.RequireVendor, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(getPartnerOrganizationsHandler)).
			With(option.Description("List all partner organizations")).
			With(option.Response(http.StatusOK, []api.PartnerOrganizationWithUsage{}))

		r.Route("/{partnerOrganizationId}", func(r chiopenapi.Router) {
			type PartnerOrganizationIDRequest struct {
				PartnerOrganizationID uuid.UUID `path:"partnerOrganizationId"`
			}

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
				r.Put("/", respond.Handler(updatePartnerOrganizationHandler)).
					With(option.Description("Update a partner organization")).
					With(option.Request(struct {
						PartnerOrganizationIDRequest
						api.CreateUpdatePartnerOrganizationRequest
					}{})).
					With(option.Response(http.StatusOK, api.PartnerOrganization{}))
				r.Delete("/", respond.Handler(deletePartnerOrganizationHandler)).
					With(option.Description("Delete a partner organization")).
					With(option.Request(PartnerOrganizationIDRequest{}))
			})
		})

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", respond.Handler(createPartnerOrganizationHandler)).
			With(option.Description("Create a new partner organization")).
			With(option.Request(api.CreateUpdatePartnerOrganizationRequest{})).
			With(option.Response(http.StatusOK, api.PartnerOrganization{}))
	})
}

var partnerOrganizationNotFound = respond.NotFound("partner organization not found")

func getPartnerOrganizationsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if partnerOrgs, err := db.GetPartnerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get partner orgs: %w", err)
	} else {
		return respond.JSON(mapping.List(partnerOrgs, mapping.PartnerOrganizationWithUsageToAPI)), nil
	}
}

func createPartnerOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if request, resp := decodeJSON[api.CreateUpdatePartnerOrganizationRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		partnerOrg := types.PartnerOrganization{
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
		}
		if err := db.CreatePartnerOrganization(ctx, &partnerOrg); err != nil {
			return nil, fmt.Errorf("failed to create partner org: %w", err)
		} else {
			return respond.JSON(mapping.PartnerOrganizationToAPI(partnerOrg)), nil
		}
	}
}

func updatePartnerOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("partnerOrganizationId")); err != nil {
		return partnerOrganizationNotFound, nil
	} else if request, resp := decodeJSON[api.CreateUpdatePartnerOrganizationRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		partnerOrg := types.PartnerOrganization{
			ID:             id,
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
		}
		if err := db.UpdatePartnerOrganization(ctx, &partnerOrg); errors.Is(err, distrerr.ErrNotFound) {
			return partnerOrganizationNotFound, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to update partner org: %w", err)
		} else {
			return respond.JSON(mapping.PartnerOrganizationToAPI(partnerOrg)), nil
		}
	}
}

//nolint:dupl
func deletePartnerOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("partnerOrganizationId")); err != nil {
		return partnerOrganizationNotFound, nil
	} else if err := db.DeletePartnerOrganizationWithID(
		ctx, id, *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return partnerOrganizationNotFound, nil
	} else if errors.Is(err, distrerr.ErrConflict) {
		return respond.Conflict("partner organization is not empty"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete partner org: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}
