package api

import (
	"github.com/distr-sh/distr/internal/types"
)

type CreateUpdateOrganizationRequest struct {
	Name                   string  `json:"name"`
	Slug                   *string `json:"slug"`
	PreConnectScript       *string `json:"preConnectScript"`
	PostConnectScript      *string `json:"postConnectScript"`
	ConnectScriptIsSudo    bool    `json:"connectScriptIsSudo"`
	ArtifactVersionMutable bool    `json:"artifactVersionMutable"`
	PrePostScriptsEnabled  bool    `json:"prePostScriptsEnabled"`
}

type OrganizationResponse struct {
	types.Organization
	SubscriptionLimits SubscriptionLimits `json:"subscriptionLimits"`
}

type OrganizationWebhookResponse struct {
	Configured bool `json:"configured"`
}

type UpdateOrganizationWebhookRequest struct {
	WebhookSecret *string `json:"webhookSecret"`
}
