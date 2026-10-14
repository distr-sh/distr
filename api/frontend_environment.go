package api

type FrontendEnvironmentResponse struct {
	SentryDSN             *string  `json:"sentryDsn,omitempty"`
	SentryEnvironment     string   `json:"sentryEnvironment,omitzero"`
	SentryTraceSampleRate *float64 `json:"sentryTraceSampleRate,omitempty"`
	PosthogToken          *string  `json:"posthogToken,omitempty"`
	PosthogAPIHost        *string  `json:"posthogApiHost,omitempty"`
	PosthogUIHost         *string  `json:"posthogUiHost,omitempty"`
	RegistryHost          string   `json:"registryHost"`
	CustomDomainTarget    *string  `json:"customDomainTarget,omitempty"`
}
