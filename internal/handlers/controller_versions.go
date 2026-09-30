package handlers

import (
	"net/http"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ControllerVersionsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Miscellaneous"))
	r.Get("/", getControllerVersionsHandler()).
		With(option.Description("List all controller versions")).
		With(option.Response(http.StatusOK, []types.ControllerVersion{}))
}

func DeprecatedAgentVersionsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Miscellaneous"))
	r.Get("/", getControllerVersionsHandler()).
		With(option.Description("Deprecated alias of /controller-versions")).
		With(option.Deprecated()).
		With(option.Response(http.StatusOK, []types.ControllerVersion{}))
}

func getControllerVersionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		if controllerVersions, err := db.GetControllerVersions(ctx); err != nil {
			log.Warn("could not get controller versions", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, controllerVersions)
		}
	}
}
