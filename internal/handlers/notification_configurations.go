package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func ApplicationNotificationConfigurationsRouter(r chiopenapi.Router) {
	notificationConfigurations[
		api.CreateUpdateApplicationNotificationConfigurationRequest,
		types.ApplicationNotificationConfiguration,
		api.ApplicationNotificationConfiguration,
	]{
		name: "application notification configuration",
		updateRequest: struct {
			notificationConfigurationIDRequest
			api.CreateUpdateApplicationNotificationConfigurationRequest
		}{},
		list:       db.GetApplicationNotificationConfigurations,
		create:     db.CreateApplicationNotificationConfiguration,
		update:     db.UpdateApplicationNotificationConfiguration,
		remove:     db.DeleteApplicationNotificationConfiguration,
		toInternal: mapping.ApplicationNotificationConfigurationToInternal,
		toAPI:      mapping.ApplicationNotificationConfigurationToAPI,
		setID:      func(config *types.ApplicationNotificationConfiguration, id uuid.UUID) { config.ID = id },
	}.mount(r)
}

func ArtifactNotificationConfigurationsRouter(r chiopenapi.Router) {
	notificationConfigurations[
		api.CreateUpdateArtifactNotificationConfigurationRequest,
		types.ArtifactNotificationConfiguration,
		api.ArtifactNotificationConfiguration,
	]{
		name: "artifact notification configuration",
		updateRequest: struct {
			notificationConfigurationIDRequest
			api.CreateUpdateArtifactNotificationConfigurationRequest
		}{},
		list:       db.GetArtifactNotificationConfigurations,
		create:     db.CreateArtifactNotificationConfiguration,
		update:     db.UpdateArtifactNotificationConfiguration,
		remove:     db.DeleteArtifactNotificationConfiguration,
		toInternal: mapping.ArtifactNotificationConfigurationToInternal,
		toAPI:      mapping.ArtifactNotificationConfigurationToAPI,
		setID:      func(config *types.ArtifactNotificationConfiguration, id uuid.UUID) { config.ID = id },
	}.mount(r)
}

type validatable interface {
	Validate() error
}

// notificationConfigurations is the CRUD of one kind of notification configuration. Application and
// artifact configurations differ only in what they are linked to, so they share these handlers.
type notificationConfigurations[Request validatable, Model any, Response any] struct {
	name string
	// updateRequest describes the body and path parameter of the update endpoint for the OpenAPI
	// spec. It is passed in because a type parameter cannot be embedded in a struct.
	updateRequest any
	list          func(ctx context.Context, orgID uuid.UUID, customerOrgID *uuid.UUID) ([]Model, error)
	create        func(ctx context.Context, config *Model) error
	update        func(ctx context.Context, config *Model) error
	remove        func(ctx context.Context, id, orgID uuid.UUID, customerOrgID *uuid.UUID) error
	toInternal    func(request Request, orgID uuid.UUID, customerOrgID *uuid.UUID) Model
	toAPI         func(config Model) Response
	setID         func(config *Model, id uuid.UUID)
}

func (h notificationConfigurations[Request, Model, Response]) mount(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature)

	var request Request
	var response Response

	r.Get("/", h.getHandler()).
		With(option.Description("list all " + h.name + "s")).
		With(option.Response(http.StatusOK, []Response{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Post("/", h.createHandler()).
		With(option.Description("create a new " + h.name)).
		With(option.Request(request)).
		With(option.Response(http.StatusOK, response))

	r.With(middleware.RequireReadWriteOrAdmin).
		Route("/{id}", func(r chiopenapi.Router) {
			r.Put("/", h.updateHandler()).
				With(option.Description("update an existing " + h.name)).
				With(option.Request(h.updateRequest)).
				With(option.Response(http.StatusOK, response))

			r.Delete("/", h.deleteHandler()).
				With(option.Description("delete an existing " + h.name)).
				With(option.Request(notificationConfigurationIDRequest{}))
		})
}

type notificationConfigurationIDRequest struct {
	ID string `path:"id"`
}

func (h notificationConfigurations[Request, Model, Response]) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		configs, err := h.list(ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if err != nil {
			respondInternalError(w, r, err, "failed to get "+h.name+"s")
			return
		}

		RespondJSON(w, mapping.List(configs, h.toAPI))
	}
}

func (h notificationConfigurations[Request, Model, Response]) createHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		request, ok := h.requestBody(w, r)
		if !ok {
			return
		}

		config := h.toInternal(request, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if err := h.create(ctx, &config); err != nil {
			respondInternalError(w, r, err, "failed to create "+h.name)
			return
		}

		RespondJSON(w, h.toAPI(config))
	}
}

func (h notificationConfigurations[Request, Model, Response]) updateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		request, ok := h.requestBody(w, r)
		if !ok {
			return
		}

		config := h.toInternal(request, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		h.setID(&config, id)
		if err := h.update(ctx, &config); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				respondInternalError(w, r, err, "failed to update "+h.name)
			}
			return
		}

		RespondJSON(w, h.toAPI(config))
	}
}

func (h notificationConfigurations[Request, Model, Response]) deleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.remove(ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID()); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				respondInternalError(w, r, err, "failed to delete "+h.name)
			}
			return
		}
	}
}

func (h notificationConfigurations[Request, Model, Response]) requestBody(
	w http.ResponseWriter,
	r *http.Request,
) (Request, bool) {
	request, err := JsonBody[Request](w, r)
	if err != nil {
		return request, false
	} else if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return request, false
	}
	return request, true
}
