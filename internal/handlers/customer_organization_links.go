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

func CustomerOrganizationLinksRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Customer Organization Links"))
	r.Use(middleware.RequireOrgAndRole)

	type customerOrganizationIDRequest struct {
		CustomerOrganizationID uuid.UUID `path:"customerOrganizationId"`
	}

	r.Get("/", getCustomerOrganizationLinksHandler()).
		With(option.Description("List all links for a customer organization")).
		With(option.Request(customerOrganizationIDRequest{})).
		With(option.Response(http.StatusOK, []api.CustomerOrganizationLink{}))

	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin)

		r.Post("/", createCustomerOrganizationLinkHandler()).
			With(option.Description("Create a link for a customer organization")).
			With(option.Request(struct {
				customerOrganizationIDRequest
				api.CreateUpdateCustomerOrganizationLinkRequest
			}{})).
			With(option.Response(http.StatusCreated, api.CustomerOrganizationLink{}))

		r.Route("/{linkId}", func(r chiopenapi.Router) {
			type linkIDRequest struct {
				customerOrganizationIDRequest
				LinkID uuid.UUID `path:"linkId"`
			}

			r.Put("/", updateCustomerOrganizationLinkHandler()).
				With(option.Description("Update a customer organization link")).
				With(option.Request(struct {
					linkIDRequest
					api.CreateUpdateCustomerOrganizationLinkRequest
				}{})).
				With(option.Response(http.StatusOK, api.CustomerOrganizationLink{}))

			r.Delete("/", deleteCustomerOrganizationLinkHandler()).
				With(option.Description("Delete a customer organization link")).
				With(option.Request(linkIDRequest{}))
		})
	})
}

func getCustomerOrganizationLinksHandler() http.HandlerFunc {
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

		links, err := db.GetCustomerOrganizationLinks(ctx, customerOrgID)
		if err != nil {
			log.Error("failed to get customer organization links", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.List(links, mapping.CustomerOrganizationLinkToAPI))
		}
	}
}

func createCustomerOrganizationLinkHandler() http.HandlerFunc {
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

		body, err := JsonBody[api.CreateUpdateCustomerOrganizationLinkRequest](w, r)
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

		link, err := db.CreateCustomerOrganizationLink(ctx, customerOrgID, body.Name, body.Link)
		if err != nil {
			log.Error("failed to create customer organization link", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusCreated)
			RespondJSON(w, mapping.CustomerOrganizationLinkToAPI(*link))
		}
	}
}

func updateCustomerOrganizationLinkHandler() http.HandlerFunc {
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

		body, err := JsonBody[api.CreateUpdateCustomerOrganizationLinkRequest](w, r)
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

		link, err := db.UpdateCustomerOrganizationLink(ctx, linkID, customerOrgID, body.Name, body.Link)
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Error("failed to update customer organization link", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			RespondJSON(w, mapping.CustomerOrganizationLinkToAPI(*link))
		}
	}
}

func deleteCustomerOrganizationLinkHandler() http.HandlerFunc {
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

		err = db.DeleteCustomerOrganizationLink(ctx, linkID, customerOrgID)
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Error("failed to delete customer organization link", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
