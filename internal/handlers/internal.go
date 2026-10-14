package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/go-chi/chi/v5"
)

func InternalRouter(r chi.Router) {
	r.Handle("/environment", respond.Handler(getFrontendEnvironmentHandler))
}

func getFrontendEnvironmentHandler(r *http.Request) (respond.Responder, error) {
	return respond.JSON(api.FrontendEnvironmentResponse{
		SentryDSN:             env.FrontendSentryDSN(),
		SentryEnvironment:     env.SentryEnvironment(),
		SentryTraceSampleRate: env.FrontendSentryTraceSampleRate(),
		PosthogToken:          env.FrontendPosthogToken(),
		PosthogAPIHost:        env.FrontendPosthogAPIHost(),
		PosthogUIHost:         env.FrontendPosthogUIHost(),
		RegistryHost:          env.RegistryHost(),
		CustomDomainTarget:    env.CustomDomainTarget(),
	}), nil
}
