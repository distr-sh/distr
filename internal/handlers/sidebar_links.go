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
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func SidebarLinksRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Sidebar Links"))
	r.Use(middleware.RequireVendor, middleware.RequireOrgAndRole)

	type customerOrganizationIDRequest struct {
		CustomerOrganizationID uuid.UUID `path:"customerOrganizationId"`
	}

	r.Get("/", getSidebarLinksHandler()).
		With(option.Description("List all sidebar links for a customer organization")).
		With(option.Request(customerOrganizationIDRequest{})).
		With(option.Response(http.StatusOK, []api.SidebarLink{}))

	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin)

		r.Post("/", createSidebarLinkHandler()).
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

			r.Put("/", updateSidebarLinkHandler()).
				With(option.Description("Update a sidebar link")).
				With(option.Request(struct {
					linkIDRequest
					api.CreateUpdateSidebarLinkRequest
				}{})).
				With(option.Response(http.StatusOK, api.SidebarLink{}))

			r.Delete("/", deleteSidebarLinkHandler()).
				With(option.Description("Delete a sidebar link")).
				With(option.Request(linkIDRequest{}))
		})
	})
}

func getSidebarLinksHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
			http.NotFound(w, r)
			return
		}

		links, err := db.GetSidebarLinks(ctx, customerOrgID)
		if err != nil {
			log.Error("failed to get sidebar links", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.List(links, mapping.SidebarLinkToAPI))
		}
	}
}

func createSidebarLinkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
			http.NotFound(w, r)
			return
		}

		body, err := JsonBody[api.CreateUpdateSidebarLinkRequest](w, r)
		if err != nil {
			return
		}

		if body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		if body.Link == "" {
			http.Error(w, "link is required", http.StatusBadRequest)
			return
		}

		link, err := db.CreateSidebarLink(ctx, *auth.CurrentOrgID(), customerOrgID, body.Name, body.Link)
		if err != nil {
			log.Error("failed to create sidebar link", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusCreated)
			RespondJSON(w, mapping.SidebarLinkToAPI(*link))
		}
	}
}

func updateSidebarLinkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		linkID, err := uuid.Parse(r.PathValue("linkId"))
		if err != nil {
			http.Error(w, "invalid link ID", http.StatusBadRequest)
			return
		}

		if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
			http.NotFound(w, r)
			return
		}

		body, err := JsonBody[api.CreateUpdateSidebarLinkRequest](w, r)
		if err != nil {
			return
		}

		if body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		if body.Link == "" {
			http.Error(w, "link is required", http.StatusBadRequest)
			return
		}

		link, err := db.UpdateSidebarLink(ctx, linkID, customerOrgID, body.Name, body.Link)
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Error("failed to update sidebar link", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			RespondJSON(w, mapping.SidebarLinkToAPI(*link))
		}
	}
}

func deleteSidebarLinkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		customerOrgID, err := uuid.Parse(r.PathValue("customerOrganizationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		linkID, err := uuid.Parse(r.PathValue("linkId"))
		if err != nil {
			http.Error(w, "invalid link ID", http.StatusBadRequest)
			return
		}

		if err := db.ValidateCustomerOrgBelongsToOrg(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
			http.NotFound(w, r)
			return
		}

		err = db.DeleteSidebarLink(ctx, linkID, customerOrgID)
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Error("failed to delete sidebar link", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
