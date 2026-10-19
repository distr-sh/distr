package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func ControllerVersionsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Get("/", respond.Handler(getControllerVersionsHandler)).
		With(option.Description("List all controller versions")).
		With(option.Response(http.StatusOK, []types.ControllerVersion{}))
}

func getControllerVersionsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	if controllerVersions, err := db.GetControllerVersions(ctx); err != nil {
		return nil, fmt.Errorf("could not get controller versions: %w", err)
	} else {
		return respond.JSON(controllerVersions), nil
	}
}
