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

func TestUserRoleParsing(t *testing.T) {
	g := NewWithT(t)

	var target struct {
		Role UserRole `json:"role"`
	}

	err := json.Unmarshal([]byte(`{"role": "read_only"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Role).To(Equal(UserRoleReadOnly))

	err = json.Unmarshal([]byte(`{"role": "read_write"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Role).To(Equal(UserRoleReadWrite))

	err = json.Unmarshal([]byte(`{"role": "admin"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Role).To(Equal(UserRoleAdmin))

	err = json.Unmarshal([]byte(`{"role": "superuser"}`), &target)
	g.Expect(err).To(HaveOccurred())
}

func TestUserRoleGreaterThan(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		a, b     UserRole
		expected bool
	}{
		{UserRoleReadOnly, UserRoleReadOnly, false},
		{UserRoleReadOnly, UserRoleReadWrite, false},
		{UserRoleReadOnly, UserRoleAdmin, false},
		{UserRoleReadWrite, UserRoleReadOnly, true},
		{UserRoleReadWrite, UserRoleReadWrite, false},
		{UserRoleReadWrite, UserRoleAdmin, false},
		{UserRoleAdmin, UserRoleReadOnly, true},
		{UserRoleAdmin, UserRoleReadWrite, true},
		{UserRoleAdmin, UserRoleAdmin, false},
	}
	for _, tc := range cases {
		g.Expect(tc.a.GreaterThan(tc.b)).
			To(Equal(tc.expected), "%q.GreaterThan(%q)", tc.a, tc.b)
	}
}

func TestUserRoleRankPanicsOnUnknown(t *testing.T) {
	g := NewWithT(t)
	g.Expect(func() { UserRole("bogus").Rank() }).To(Panic())
	g.Expect(func() { UserRole("").Rank() }).To(Panic())
}

func TestFeaturesForSubscriptionType(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		subscriptionType SubscriptionType
		expected         []Feature
	}{
		{SubscriptionTypeCommunity, []Feature{}},
		{SubscriptionTypeTrial, []Feature{FeatureLicensing}},
		{SubscriptionTypePro, []Feature{FeatureLicensing}},
		{
			SubscriptionTypeBusiness,
			[]Feature{FeatureLicensing, FeaturePartnerManagement, FeatureVulnerabilities},
		},
		{
			SubscriptionTypeEnterprise,
			[]Feature{FeatureLicensing, FeaturePartnerManagement, FeatureVulnerabilities},
		},
	}
	for _, tc := range cases {
		g.Expect(FeaturesForSubscriptionType(tc.subscriptionType)).
			To(ConsistOf(tc.expected), "features for %q", tc.subscriptionType)
	}
}

func TestEnterpriseIncludesAllBusinessFeatures(t *testing.T) {
	g := NewWithT(t)
	g.Expect(FeaturesForSubscriptionType(SubscriptionTypeEnterprise)).
		To(ContainElements(FeaturesForSubscriptionType(SubscriptionTypeBusiness)))
}

func TestDeploymentTypeParsing(t *testing.T) {
	g := NewWithT(t)

	var target struct {
		Type DeploymentType `json:"type"`
	}

	err := json.Unmarshal([]byte(`{"type": "docker"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentTypeDocker))

	err = json.Unmarshal([]byte(`{"type": "kubernetes"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentTypeKubernetes))

	err = json.Unmarshal([]byte(`{"type": "swarm"}`), &target)
	g.Expect(err).To(MatchError(ErrInvalidDeploymentType))
}
