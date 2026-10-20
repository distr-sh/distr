package api

import (
	"regexp"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
)

type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

type UpdateOrganizationRequest struct {
	Name                        string  `json:"name"`
	Slug                        *string `json:"slug"`
	DockerPreConnectScript      *string `json:"dockerPreConnectScript"`
	DockerPostConnectScript     *string `json:"dockerPostConnectScript"`
	DockerConnectScriptIsSudo   bool    `json:"dockerConnectScriptIsSudo"`
	KubernetesPreConnectScript  *string `json:"kubernetesPreConnectScript"`
	KubernetesPostConnectScript *string `json:"kubernetesPostConnectScript"`
	ArtifactVersionMutable      bool    `json:"artifactVersionMutable"`
	PrePostScriptsEnabled       bool    `json:"prePostScriptsEnabled"`
	ControllerImageRepository   *string `json:"controllerImageRepository"`
}

func (r *UpdateOrganizationRequest) Normalize() {
	if r.ControllerImageRepository == nil {
		return
	}
	if value := validation.NormalizeImageRepository(*r.ControllerImageRepository); value != "" {
		r.ControllerImageRepository = &value
	} else {
		r.ControllerImageRepository = nil
	}
}

func (r *UpdateOrganizationRequest) ValidateControllerImageRepository() error {
	if r.ControllerImageRepository == nil {
		return nil
	}
	return validation.ValidateImageRepository(*r.ControllerImageRepository)
}

type OrganizationResponse struct {
	types.Organization
	SubscriptionLimits               SubscriptionLimits `json:"subscriptionLimits"`
	CurrentBillableUserAccountCount  int64              `json:"currentBillableUserAccountCount"`
	CurrentCustomerOrganizationCount int64              `json:"currentCustomerOrganizationCount"`
	ControllerImageRepository        *string            `json:"controllerImageRepository,omitempty"`
	DefaultControllerImageRepository string             `json:"defaultControllerImageRepository"`
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
