package types

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/distr-sh/distr/internal/openapischema"
	"github.com/oaswrap/spec/openapi"
)

var ErrInvalidCustomerOrganizationFeature = fmt.Errorf("invalid customer organization feature")

type CustomerOrganizationFeature string

const (
	CustomerOrganizationFeatureDeploymentTargets CustomerOrganizationFeature = "deployment_targets"
	CustomerOrganizationFeatureArtifacts         CustomerOrganizationFeature = "artifacts"
	CustomerOrganizationFeatureAlerts            CustomerOrganizationFeature = "alerts"
	// CustomerOrganizationFeatureUpdateNotifications lets the customer organization configure who
	// of its own users hears about a new application or artifact version.
	CustomerOrganizationFeatureUpdateNotifications CustomerOrganizationFeature = "update_notifications"
	CustomerOrganizationFeatureSupportBundles      CustomerOrganizationFeature = "support_bundles"
	// CustomerOrganizationFeatureSettings gives admins of the customer organization a settings page,
	// on which they configure e.g. the controller image repository of their deployment targets.
	CustomerOrganizationFeatureSettings CustomerOrganizationFeature = "settings"
	// CustomerOrganizationFeatureOidcProviders lets admins of the customer organization configure
	// their own portal domain and their own OIDC providers on it. It requires
	// CustomerOrganizationFeatureSettings, since both are configured on the settings page.
	CustomerOrganizationFeatureOidcProviders CustomerOrganizationFeature = "oidc_providers"
	// CustomerOrganizationFeatureVulnerabilities shares its name with FeatureVulnerabilities, whose
	// customer side it gates.
	CustomerOrganizationFeatureVulnerabilities CustomerOrganizationFeature = "vulnerabilities"
)

func (CustomerOrganizationFeature) OpenAPISchema() *openapi.Schema {
	return openapischema.StringEnum(
		CustomerOrganizationFeatureDeploymentTargets,
		CustomerOrganizationFeatureArtifacts,
		CustomerOrganizationFeatureAlerts,
		CustomerOrganizationFeatureUpdateNotifications,
		CustomerOrganizationFeatureSupportBundles,
		CustomerOrganizationFeatureSettings,
		CustomerOrganizationFeatureOidcProviders,
		CustomerOrganizationFeatureVulnerabilities,
	)
}

// CustomerOrganizationFeatureRequirements maps each customer feature that an organization can only
// hand out with an organization feature of its own to that feature. The organization feature
// decides, not the plan. It is mirrored by CUSTOMER_FEATURE_REQUIREMENTS in
// frontend/ui/src/app/types/customer-features.ts. Change both together.
var CustomerOrganizationFeatureRequirements = map[CustomerOrganizationFeature]Feature{
	CustomerOrganizationFeatureAlerts:              FeatureNotifications,
	CustomerOrganizationFeatureUpdateNotifications: FeatureNotifications,
	CustomerOrganizationFeatureSupportBundles:      FeatureSupport,
	CustomerOrganizationFeatureVulnerabilities:     FeatureVulnerabilities,
	CustomerOrganizationFeatureOidcProviders:       FeatureCustomOidcProviders,
	CustomerOrganizationFeatureSettings:            FeatureCustomerSettings,
}

type GrantedCustomerOrganizationFeature struct {
	Feature         Feature
	CustomerFeature CustomerOrganizationFeature
}

// GrantedCustomerOrganizationFeatures are the customer features every customer of an organization
// receives with an organization feature. They are granted when a customer is created and when the
// organization gains the feature, never on later updates, so that a vendor can switch them off for
// single customers.
var GrantedCustomerOrganizationFeatures = []GrantedCustomerOrganizationFeature{
	{FeatureVulnerabilities, CustomerOrganizationFeatureVulnerabilities},
}

// DefaultCustomerOrganizationFeatures returns the features of a customer that org creates.
func DefaultCustomerOrganizationFeatures(org *Organization) []CustomerOrganizationFeature {
	features := []CustomerOrganizationFeature{
		CustomerOrganizationFeatureDeploymentTargets,
		CustomerOrganizationFeatureArtifacts,
	}
	for _, granted := range GrantedCustomerOrganizationFeatures {
		if org.HasFeature(granted.Feature) {
			features = append(features, granted.CustomerFeature)
		}
	}
	return features
}

// GainedCustomerOrganizationFeatures returns the customer features that come with the organization
// features in features but not in previousFeatures.
func GainedCustomerOrganizationFeatures(previousFeatures, features []Feature) []CustomerOrganizationFeature {
	var gained []CustomerOrganizationFeature
	for _, granted := range GrantedCustomerOrganizationFeatures {
		if slices.Contains(features, granted.Feature) && !slices.Contains(previousFeatures, granted.Feature) {
			gained = append(gained, granted.CustomerFeature)
		}
	}
	return gained
}

func ParseCustomerOrganizationFeature(value string) (CustomerOrganizationFeature, error) {
	switch value {
	case string(CustomerOrganizationFeatureDeploymentTargets):
		return CustomerOrganizationFeatureDeploymentTargets, nil
	case string(CustomerOrganizationFeatureArtifacts):
		return CustomerOrganizationFeatureArtifacts, nil
	case string(CustomerOrganizationFeatureAlerts):
		return CustomerOrganizationFeatureAlerts, nil
	case string(CustomerOrganizationFeatureUpdateNotifications):
		return CustomerOrganizationFeatureUpdateNotifications, nil
	case string(CustomerOrganizationFeatureSupportBundles):
		return CustomerOrganizationFeatureSupportBundles, nil
	case string(CustomerOrganizationFeatureSettings):
		return CustomerOrganizationFeatureSettings, nil
	case string(CustomerOrganizationFeatureOidcProviders):
		return CustomerOrganizationFeatureOidcProviders, nil
	case string(CustomerOrganizationFeatureVulnerabilities):
		return CustomerOrganizationFeatureVulnerabilities, nil
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
