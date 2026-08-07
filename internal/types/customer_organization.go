package types

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCustomerOrganizationFeature = fmt.Errorf("invalid customer organization feature")

type CustomerOrganizationFeature string

const (
	CustomerOrganizationFeatureDeploymentTargets CustomerOrganizationFeature = "deployment_targets"
	CustomerOrganizationFeatureArtifacts         CustomerOrganizationFeature = "artifacts"
	CustomerOrganizationFeatureAlerts            CustomerOrganizationFeature = "alerts"
	CustomerOrganizationFeatureSupportBundles    CustomerOrganizationFeature = "support_bundles"
	// CustomerOrganizationFeatureOidcProviders lets admins of the customer organization
	// configure their own portal domain and their own OIDC providers on it.
	CustomerOrganizationFeatureOidcProviders CustomerOrganizationFeature = "oidc_providers"
)

func ParseCustomerOrganizationFeature(value string) (CustomerOrganizationFeature, error) {
	switch value {
	case string(CustomerOrganizationFeatureDeploymentTargets):
		return CustomerOrganizationFeatureDeploymentTargets, nil
	case string(CustomerOrganizationFeatureArtifacts):
		return CustomerOrganizationFeatureArtifacts, nil
	case string(CustomerOrganizationFeatureAlerts):
		return CustomerOrganizationFeatureAlerts, nil
	case string(CustomerOrganizationFeatureSupportBundles):
		return CustomerOrganizationFeatureSupportBundles, nil
	case string(CustomerOrganizationFeatureOidcProviders):
		return CustomerOrganizationFeatureOidcProviders, nil
	default:
		return "", fmt.Errorf("%w: %v", ErrInvalidCustomerOrganizationFeature, value)
	}
}

func (ref *CustomerOrganizationFeature) UnmarshalJSON(data []byte) error {
	var featureStr string
	if err := json.Unmarshal(data, &featureStr); err != nil {
		return err
	} else if feature, err := ParseCustomerOrganizationFeature(featureStr); err != nil {
		return err
	} else {
		*ref = feature
		return nil
	}
}

type CustomerOrganization struct {
	ID                    uuid.UUID                     `db:"id" json:"id"`
	CreatedAt             time.Time                     `db:"created_at" json:"createdAt"`
	OrganizationID        uuid.UUID                     `db:"organization_id" json:"organizationId"`
	ImageID               *uuid.UUID                    `db:"image_id" json:"imageId,omitempty"`
	Name                  string                        `db:"name" json:"name"`
	Features              []CustomerOrganizationFeature `db:"features" json:"features"`
	PartnerOrganizationID *uuid.UUID                    `db:"partner_organization_id" json:"partnerOrganizationId,omitempty"` //nolint:lll
}

func (co *CustomerOrganization) HasFeature(feature CustomerOrganizationFeature) bool {
	return slices.Contains(co.Features, feature)
}

type CustomerOrganizationWithUsage struct {
	CustomerOrganization
	UserCount             int64 `db:"user_count" json:"userCount"`
	DeploymentTargetCount int64 `db:"deployment_target_count" json:"deploymentTargetCount"`
}
