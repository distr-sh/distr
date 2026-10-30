package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/go-chi/chi/v5"
)

type frontendEnvironmentResponse struct {
	SentryDSN             *string  `json:"sentryDsn,omitempty"`
	SentryEnvironment     string   `json:"sentryEnvironment,omitzero"`
	SentryTraceSampleRate *float64 `json:"sentryTraceSampleRate,omitempty"`
	PosthogToken          *string  `json:"posthogToken,omitempty"`
	PosthogAPIHost        *string  `json:"posthogApiHost,omitempty"`
	PosthogUIHost         *string  `json:"posthogUiHost,omitempty"`
	RegistryHost          string   `json:"registryHost"`
	CustomDomainTarget    *string  `json:"customDomainTarget,omitempty"`
}

func InternalRouter(r chi.Router) {
	r.Handle("/environment", respond.Handler(getFrontendEnvironmentHandler))
}

func getFrontendEnvironmentHandler(r *http.Request) (respond.Responder, error) {
	return respond.JSON(frontendEnvironmentResponse{
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
