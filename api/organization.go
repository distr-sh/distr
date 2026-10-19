package api

import (
	"regexp"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type CreateUpdateOrganizationRequest struct {
	Name                        string  `json:"name"`
	Slug                        *string `json:"slug"`
	DockerPreConnectScript      *string `json:"dockerPreConnectScript"`
	DockerPostConnectScript     *string `json:"dockerPostConnectScript"`
	DockerConnectScriptIsSudo   bool    `json:"dockerConnectScriptIsSudo"`
	KubernetesPreConnectScript  *string `json:"kubernetesPreConnectScript"`
	KubernetesPostConnectScript *string `json:"kubernetesPostConnectScript"`
	ArtifactVersionMutable      bool    `json:"artifactVersionMutable"`
	PrePostScriptsEnabled       bool    `json:"prePostScriptsEnabled"`
}

type CreateOrganizationRequest struct {
	CreateUpdateOrganizationRequest
	AdminEmail string `json:"adminEmail,omitempty"`
}

type CreateOrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	InviteURL string    `json:"inviteUrl,omitempty"`
}

type OrganizationResponse struct {
	types.Organization
	SubscriptionLimits               SubscriptionLimits `json:"subscriptionLimits"`
	CurrentBillableUserAccountCount  int64              `json:"currentBillableUserAccountCount"`
	CurrentCustomerOrganizationCount int64              `json:"currentCustomerOrganizationCount"`
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
