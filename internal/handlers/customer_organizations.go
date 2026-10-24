package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func CustomerOrganizationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Customers"))
	r.With(middleware.RequireVendorOrPartner, middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(getCustomerOrganizationsHandler)).
			With(option.Description("List all customer organizations")).
			With(option.Response(http.StatusOK, []api.CustomerOrganizationWithUsage{}))

		r.Route("/{customerOrganizationId}", func(r chiopenapi.Router) {
			type CustomerOrganizationIDRequest struct {
				CustomerOrganizationID uuid.UUID `path:"customerOrganizationId"`
			}

			r.Route("/links", SidebarLinksRouter)

			r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
				r.With(middleware.RequireVendorOrPartner).Group(func(r chiopenapi.Router) {
					r.With(middleware.RequireReadWriteOrAdmin).
						Put("/", respond.Handler(updateCustomerOrganizationHandler)).
						With(option.Description("Update a customer organization")).
						With(option.Request(struct {
							CustomerOrganizationIDRequest
							api.CreateUpdateCustomerOrganizationRequest
						}{})).
						With(option.Response(http.StatusOK, api.CustomerOrganization{}))

					r.With(middleware.RequireAdmin).
						Delete("/", respond.Handler(deleteCustomerOrganizationHandler)).
						With(option.Description("Delete a customer organization")).
						With(option.Request(CustomerOrganizationIDRequest{}))
				})

				r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.PartnerManagementFeatureMiddleware).
					Put("/partner", respond.Handler(assignCustomerToPartnerHandler)).
					With(option.Description("Assign or unassign a partner organization for a customer organization")).
					With(option.Request(struct {
						CustomerOrganizationIDRequest
						api.AssignCustomerToPartnerRequest
					}{})).
					With(option.Response(http.StatusOK, api.CustomerOrganization{}))
			})
		})

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", respond.Handler(createCustomerOrganizationHandler)).
			With(option.Description("Create a new customer organization")).
			With(option.Request(api.CreateUpdateCustomerOrganizationRequest{})).
			With(option.Response(http.StatusOK, api.CustomerOrganization{}))
	})
}

func getCustomerOrganizationsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	var customerOrganizations []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		customerOrganizations, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		customerOrganizations, err = db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get customer orgs: %w", err)
	} else {
		return respond.JSON(mapping.List(customerOrganizations, mapping.CustomerOrganizationWithUsageToAPI)), nil
	}
}

func createCustomerOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if request, resp := decodeJSON[api.CreateUpdateCustomerOrganizationRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		customerOrganization := types.CustomerOrganization{
			OrganizationID:        *auth.CurrentOrgID(),
			Name:                  request.Name,
			ImageID:               request.ImageID,
			PartnerOrganizationID: auth.CurrentPartnerOrgID(),
		}

		if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
			if limitReached, err := subscription.IsCustomerOrganizationLimitReached(ctx, *auth.CurrentOrg()); err != nil {
				return nil, fmt.Errorf("failed to check customer limit: %w", err)
			} else if limitReached {
				return respond.Forbidden("customer limit reached"), nil
			} else if err := db.CreateCustomerOrganization(ctx, &customerOrganization); errors.Is(err, distrerr.ErrBadRequest) {
				return respond.BadRequest(err.Error()), nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to create customer org: %w", err)
			} else {
				return nil, nil
			}
		}); resp != nil || err != nil {
			return resp, err
		} else {
			return respond.JSON(mapping.CustomerOrganizationToAPI(customerOrganization)), nil
		}
	}
}

func updateCustomerOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return customerOrganizationNotFound, nil
	} else if request, resp := decodeJSON[api.CreateUpdateCustomerOrganizationRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if resp, err := checkReachableByPartner(ctx, &id, customerOrganizationNotFound); resp != nil || err != nil {
		return resp, err
	} else {
		var features []types.CustomerOrganizationFeature
		if request.Features == nil {
			features = []types.CustomerOrganizationFeature{
				types.CustomerOrganizationFeatureDeploymentTargets,
				types.CustomerOrganizationFeatureArtifacts,
			}
		} else {
			features = request.Features
		}

		// The customer's providers are configured on the vendor's own plan, so the vendor cannot hand
		// out a feature it does not have itself. This cannot live in request.Validate(), which sees
		// neither the context nor the organization.
		if slices.Contains(features, types.CustomerOrganizationFeatureOidcProviders) &&
			!auth.CurrentOrg().HasFeature(types.FeatureCustomOidcProviders) {
			return respond.Forbidden("the custom identity provider feature is not available on your plan"), nil
		}

		if resp, err := checkProCustomerOrganizationFeatures(ctx, id, features); resp != nil || err != nil {
			return resp, err
		}

		customerOrganization := types.CustomerOrganization{
			ID:             id,
			OrganizationID: *auth.CurrentOrgID(),
			Name:           request.Name,
			ImageID:        request.ImageID,
			Features:       features,
		}

		if err := db.UpdateCustomerOrganization(ctx, &customerOrganization); errors.Is(err, distrerr.ErrNotFound) {
			return customerOrganizationNotFound, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to update customer org: %w", err)
		} else {
			return respond.JSON(mapping.CustomerOrganizationToAPI(customerOrganization)), nil
		}
	}
}

// checkProCustomerOrganizationFeatures rejects adding a Pro customer feature without the Pro plan.
// Features granted before a downgrade may stay, so that the customer can still be edited.
func checkProCustomerOrganizationFeatures(
	ctx context.Context,
	id uuid.UUID,
	features []types.CustomerOrganizationFeature,
) (respond.Responder, error) {
	auth := auth.Authentication.Require(ctx)
	if auth.CurrentOrg().SubscriptionType.IsPro() ||
		!slices.ContainsFunc(features, func(f types.CustomerOrganizationFeature) bool {
			return slices.Contains(types.ProCustomerOrganizationFeatures, f)
		}) {
		return nil, nil
	}
	existing, err := db.GetCustomerOrganizationByID(ctx, id)
	if errors.Is(err, distrerr.ErrNotFound) || (err == nil && existing.OrganizationID != *auth.CurrentOrgID()) {
		return customerOrganizationNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get customer org: %w", err)
	}
	for _, feature := range features {
		if slices.Contains(types.ProCustomerOrganizationFeatures, feature) && !existing.HasFeature(feature) {
			return respond.Forbidden("alerts, update notifications and support bundles require the Pro plan"), nil
		}
	}
	return nil, nil
}

func assignCustomerToPartnerHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return customerOrganizationNotFound, nil
	} else if request, resp := decodeJSON[api.AssignCustomerToPartnerRequest](r); resp != nil {
		return resp, nil
	} else {
		if request.PartnerOrganizationID != nil {
			if err := db.ValidatePartnerOrgBelongsToOrg(
				ctx, *request.PartnerOrganizationID, *auth.CurrentOrgID(),
			); err != nil {
				if errors.Is(err, db.ErrPartnerOrgNotInOrg) {
					return respond.BadRequest("partner organization not found"), nil
				} else {
					return nil, fmt.Errorf("failed to check partner in org: %w", err)
				}
			}
		}

		if err := db.SetCustomerOrganizationPartner(
			ctx, id, *auth.CurrentOrgID(), request.PartnerOrganizationID,
		); errors.Is(err, distrerr.ErrNotFound) {
			return customerOrganizationNotFound, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to assign partner to customer org: %w", err)
		} else if customerOrg, err := db.GetCustomerOrganizationByID(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to get customer org after partner assignment: %w", err)
		} else {
			return respond.JSON(mapping.CustomerOrganizationToAPI(customerOrg.CustomerOrganization)), nil
		}
	}
}

//nolint:dupl
func deleteCustomerOrganizationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return customerOrganizationNotFound, nil
	} else {
		if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
			if co, err := db.GetCustomerOrganizationByID(ctx, id); err != nil {
				if errors.Is(err, distrerr.ErrNotFound) {
					return customerOrganizationNotFound, nil
				} else {
					return nil, fmt.Errorf("failed to get customer org: %w", err)
				}
			} else if !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
				return customerOrganizationNotFound, nil
			}
		}

		if err := db.DeleteCustomerOrganizationWithID(
			ctx, id, *auth.CurrentOrgID(),
		); errors.Is(err, distrerr.ErrNotFound) {
			return customerOrganizationNotFound, nil
		} else if errors.Is(err, distrerr.ErrConflict) {
			return respond.Conflict("customer is not empty"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to delete customer org: %w", err)
		} else {
			return respond.NoContent(), nil
		}
	}
}
