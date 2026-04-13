package api

import (
	"fmt"
	"strings"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
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

func (r UpdateOrganizationWebhookRequest) Validate() error {
	if r.WebhookSecret != nil {
		if strings.TrimSpace(*r.WebhookSecret) == "" {
			return validation.NewValidationFailedError(fmt.Sprintf("invalid webhookSecret: \"%v\"", *r.WebhookSecret))
		}
	}

	return nil
}
