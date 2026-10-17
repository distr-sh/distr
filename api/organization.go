package api

import (
	"regexp"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
)

type CreateUpdateOrganizationRequest struct {
	Name                      string  `json:"name"`
	Slug                      *string `json:"slug"`
	PreConnectScript          *string `json:"preConnectScript"`
	PostConnectScript         *string `json:"postConnectScript"`
	ConnectScriptIsSudo       bool    `json:"connectScriptIsSudo"`
	ArtifactVersionMutable    bool    `json:"artifactVersionMutable"`
	PrePostScriptsEnabled     bool    `json:"prePostScriptsEnabled"`
	ControllerImageRepository *string `json:"controllerImageRepository"`
}

func (r *CreateUpdateOrganizationRequest) Normalize() {
	if r.ControllerImageRepository == nil {
		return
	}
	if value := validation.NormalizeImageRepository(*r.ControllerImageRepository); value != "" {
		r.ControllerImageRepository = &value
	} else {
		r.ControllerImageRepository = nil
	}
}

func (r *CreateUpdateOrganizationRequest) ValidateControllerImageRepository() error {
	if r.ControllerImageRepository == nil {
		return nil
	}
	return validation.ValidateImageRepository(*r.ControllerImageRepository)
}

type OrganizationResponse struct {
	types.Organization
	SubscriptionLimits                     SubscriptionLimits                    `json:"subscriptionLimits"`
	CurrentBillableUserAccountCount        int64                                 `json:"currentBillableUserAccountCount"`
	CurrentCustomerOrganizationCount       int64                                 `json:"currentCustomerOrganizationCount"`
	ControllerImageRepository              *string                               `json:"controllerImageRepository,omitempty"` //nolint:lll
	DefaultControllerImageRepository       string                                `json:"defaultControllerImageRepository"`
	DefaultControllerImageRepositorySource types.ControllerImageRepositorySource `json:"defaultControllerImageRepositorySource"`   //nolint:lll
	ControllerImageCustomDomain            bool                                  `json:"controllerImageRepositoryUseCustomDomain"` //nolint:lll
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
