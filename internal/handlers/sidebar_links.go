package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func SidebarLinksRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Sidebar Links"))
	r.Use(middleware.RequireVendorOrPartner, middleware.RequireOrgAndRole)

	type customerOrganizationIDRequest struct {
		CustomerOrganizationID uuid.UUID `path:"customerOrganizationId"`
	}

	r.Get("/", respond.Handler(getSidebarLinksHandler)).
		With(option.Description("List all sidebar links for a customer organization")).
		With(option.Request(customerOrganizationIDRequest{})).
		With(option.Response(http.StatusOK, []api.SidebarLink{}))

	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin)

		r.Post("/", respond.Handler(createSidebarLinkHandler)).
			With(option.Description("Create a sidebar link for a customer organization")).
			With(option.Request(struct {
				customerOrganizationIDRequest
				api.CreateUpdateSidebarLinkRequest
			}{})).
			With(option.Response(http.StatusCreated, api.SidebarLink{}))

		r.Route("/{linkId}", func(r chiopenapi.Router) {
			type linkIDRequest struct {
				customerOrganizationIDRequest
				LinkID uuid.UUID `path:"linkId"`
			}

			r.Put("/", respond.Handler(updateSidebarLinkHandler)).
				With(option.Description("Update a sidebar link")).
				With(option.Request(struct {
					linkIDRequest
					api.CreateUpdateSidebarLinkRequest
				}{})).
				With(option.Response(http.StatusOK, api.SidebarLink{}))

			r.Delete("/", respond.Handler(deleteSidebarLinkHandler)).
				With(option.Description("Delete a sidebar link")).
				With(option.Request(linkIDRequest{}))
		})
	})
}

var (
	sidebarLinkCustomerNotFound = respond.NotFound("customer not found")
	sidebarLinkNotFound         = respond.NotFound("sidebar link not found")
)

func validatePartnerAccessToCustomer(ctx context.Context, customerOrgID uuid.UUID) (bool, error) {
	a := auth.Authentication.Require(ctx)
	partnerOrgID := a.CurrentPartnerOrgID()
	if partnerOrgID == nil {
		return true, nil
	}
	co, err := db.GetCustomerOrganizationByID(ctx, customerOrgID)
	if err != nil {
		return false, err
	}
	return util.PtrEq(co.PartnerOrganizationID, partnerOrgID), nil
}

func getSidebarLinksHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if ok, err := validatePartnerAccessToCustomer(ctx, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to validate partner access: %w", err)
	} else if !ok {
		return sidebarLinkCustomerNotFound, nil
	} else if links, err := db.GetSidebarLinks(ctx, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to get sidebar links: %w", err)
	} else {
		return respond.JSON(mapping.List(links, mapping.SidebarLinkToAPI)), nil
	}
}

func createSidebarLinkHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if ok, err := validatePartnerAccessToCustomer(ctx, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to validate partner access: %w", err)
	} else if !ok {
		return sidebarLinkCustomerNotFound, nil
	} else if body, resp := decodeJSON[api.CreateUpdateSidebarLinkRequest](r); resp != nil {
		return resp, nil
	} else if body.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if body.Link == "" {
		return respond.BadRequest("link is required"), nil
	} else if link, err := db.CreateSidebarLink(
		ctx, *auth.CurrentOrgID(), customerOrgID, body.Name, body.Link,
	); err != nil {
		return nil, fmt.Errorf("failed to create sidebar link: %w", err)
	} else {
		return respond.JSON(mapping.SidebarLinkToAPI(*link)).WithStatus(http.StatusCreated), nil
	}
}

func updateSidebarLinkHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if linkID, err := uuid.Parse(r.PathValue("linkId")); err != nil {
		return respond.BadRequest("invalid link ID"), nil
	} else if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if ok, err := validatePartnerAccessToCustomer(ctx, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to validate partner access: %w", err)
	} else if !ok {
		return sidebarLinkCustomerNotFound, nil
	} else if body, resp := decodeJSON[api.CreateUpdateSidebarLinkRequest](r); resp != nil {
		return resp, nil
	} else if body.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if body.Link == "" {
		return respond.BadRequest("link is required"), nil
	} else if link, err := db.UpdateSidebarLink(ctx, linkID, customerOrgID, body.Name, body.Link); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return sidebarLinkNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to update sidebar link: %w", err)
		}
	} else {
		return respond.JSON(mapping.SidebarLinkToAPI(*link)), nil
	}
}

func deleteSidebarLinkHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId")); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if linkID, err := uuid.Parse(r.PathValue("linkId")); err != nil {
		return respond.BadRequest("invalid link ID"), nil
	} else if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
		return sidebarLinkCustomerNotFound, nil
	} else if ok, err := validatePartnerAccessToCustomer(ctx, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to validate partner access: %w", err)
	} else if !ok {
		return sidebarLinkCustomerNotFound, nil
	} else if err := db.DeleteSidebarLink(ctx, linkID, customerOrgID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return sidebarLinkNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to delete sidebar link: %w", err)
		}
	} else {
		return respond.NoContent(), nil
	}
}
