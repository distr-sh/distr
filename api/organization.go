package api

import (
	"regexp"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

// CreateOrganizationRequest only exposes the fields that are actually persisted when an organization is
// created. Other settings (scripts, portal customization, etc.) are configured afterwards via an update.
type CreateOrganizationRequest struct {
	Name string  `json:"name"`
	Slug *string `json:"slug"`
}

type UpdateOrganizationRequest struct {
	Name                   string     `json:"name"`
	Slug                   *string    `json:"slug"`
	PreConnectScript       *string    `json:"preConnectScript"`
	PostConnectScript      *string    `json:"postConnectScript"`
	ConnectScriptIsSudo    bool       `json:"connectScriptIsSudo"`
	ArtifactVersionMutable bool       `json:"artifactVersionMutable"`
	PrePostScriptsEnabled  bool       `json:"prePostScriptsEnabled"`
	PageTitle              *string    `json:"pageTitle"`
	FaviconImageID         *uuid.UUID `json:"faviconImageId"`
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

// PortalResponse contains the host-resolved portal branding (browser tab title and favicon) that applies to
// everyone visiting an organization's custom app domain, regardless of authentication.
type PortalResponse struct {
	PageTitle  *string `json:"pageTitle,omitempty"`
	FaviconUrl *string `json:"faviconUrl,omitempty"`
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
