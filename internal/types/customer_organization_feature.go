package types

import (
	"encoding/json"
	"fmt"
	"slices"
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
	// CustomerOrganizationFeatureOidcProviders lets admins of the customer organization
	// configure their own portal domain and their own OIDC providers on it.
	CustomerOrganizationFeatureOidcProviders CustomerOrganizationFeature = "oidc_providers"
	// CustomerOrganizationFeatureVulnerabilities shares its name with FeatureVulnerabilities, whose
	// customer side it gates.
	CustomerOrganizationFeatureVulnerabilities CustomerOrganizationFeature = "vulnerabilities"
)

// ProCustomerOrganizationFeatures can only be granted by an organization with a pro subscription, since
// the routes behind them are gated with middleware.ProFeature.
var ProCustomerOrganizationFeatures = []CustomerOrganizationFeature{
	CustomerOrganizationFeatureAlerts,
	CustomerOrganizationFeatureUpdateNotifications,
	CustomerOrganizationFeatureSupportBundles,
}

// BusinessCustomerOrganizationFeatures maps each Business customer feature to the organization
// feature that unlocks it. The organization feature decides, not the plan.
var BusinessCustomerOrganizationFeatures = map[CustomerOrganizationFeature]Feature{
	CustomerOrganizationFeatureVulnerabilities: FeatureVulnerabilities,
	CustomerOrganizationFeatureOidcProviders:   FeatureCustomOidcProviders,
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
