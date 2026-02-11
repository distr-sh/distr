package types

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
)

func TestDeploymentStatusTypeParsing(t *testing.T) {
	g := NewWithT(t)

	var target struct {
		Type DeploymentStatusType `json:"type"`
	}

	err := json.Unmarshal([]byte(`{"type": "healthy"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentStatusTypeHealthy))

	err = json.Unmarshal([]byte(`{"type": "ok"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentStatusTypeRunning))

	err = json.Unmarshal([]byte(`{"type": "does-not-exist"}`), &target)
	g.Expect(err).To(MatchError(ErrInvalidDeploymentStatusType))
}

func TestParseCustomerOrganizationFeature(t *testing.T) {
	g := NewWithT(t)

	// Test valid features
	feature, err := ParseCustomerOrganizationFeature("deployment_targets")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(feature).To(Equal(CustomerOrganizationFeatureDeploymentTargets))

	feature, err = ParseCustomerOrganizationFeature("artifacts")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(feature).To(Equal(CustomerOrganizationFeatureArtifacts))

	feature, err = ParseCustomerOrganizationFeature("alerts")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(feature).To(Equal(CustomerOrganizationFeatureAlerts))

	// Test invalid feature
	_, err = ParseCustomerOrganizationFeature("invalid")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(Equal("invalid customer organization feature"))
}
