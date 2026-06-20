package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/featureflags"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func ExperimentalFeatureFlagsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Feature Flags"))
	r.With(middleware.RequireOrgAndRole, middleware.RequireAdmin).
		Get("/", getExperimentalFeatureFlagsHandler()).
		With(option.Description("List experimental feature flags and their enabled state for this Hub instance")).
		With(option.Response(http.StatusOK, []api.ExperimentalFeatureFlag{}))
}

func getExperimentalFeatureFlagsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := featureflags.NewRegistry(env.ExperimentalFeatureFlags())
		RespondJSON(w, experimentalFeatureFlagResponses(registry.Flags()))
	}
}

func experimentalFeatureFlagResponses(flags []featureflags.Flag) []api.ExperimentalFeatureFlag {
	responses := make([]api.ExperimentalFeatureFlag, 0, len(flags))
	for _, flag := range flags {
		responses = append(responses, api.ExperimentalFeatureFlag{
			Key:         string(flag.Key),
			Label:       flag.Label,
			Description: flag.Description,
			Milestone:   flag.Milestone,
			Enabled:     flag.Enabled,
		})
	}
	return responses
}
