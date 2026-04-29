package api

import (
	"regexp"

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
	OpenTofuEnabled        bool    `json:"openTofuEnabled"`
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
		if ok, err := regexp.MatchString("^whsec_[A-Za-z0-9]{1,128}$", *r.WebhookSecret); err != nil {
			return err
		} else if !ok {
			return validation.NewValidationFailedError("invalid webhookSecret format")
		}
	}

	return nil
}
