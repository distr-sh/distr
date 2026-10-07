package notification

import (
	"slices"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestDeploymentsBehind(t *testing.T) {
	base := time.Now()
	v200 := types.ApplicationVersion{ID: uuid.New(), Name: "2.0.0", CreatedAt: base}
	v210 := types.ApplicationVersion{ID: uuid.New(), Name: "2.1.0", CreatedAt: base.Add(time.Hour)}
	// A bugfix for the 2.0 line, released after 2.1.0.
	v205 := types.ApplicationVersion{ID: uuid.New(), Name: "2.0.5", CreatedAt: base.Add(2 * time.Hour)}
	nightly := types.ApplicationVersion{ID: uuid.New(), Name: "nightly", CreatedAt: base.Add(-time.Hour)}
	semverOnly := []types.ApplicationVersion{v200, v210, v205}

	deployments := []types.DeploymentPendingUpdate{
		{DeploymentTargetName: "on-2.0.0", CurrentVersionID: v200.ID},
		{DeploymentTargetName: "on-2.1.0", CurrentVersionID: v210.ID},
	}

	tests := []struct {
		name     string
		strategy types.VersioningStrategy
		versions []types.ApplicationVersion
		want     []string
	}{
		{
			name:     "semver",
			strategy: types.VersioningStrategySemver,
			versions: semverOnly,
			want:     []string{"on-2.0.0"},
		},
		{
			name:     "the bugfix is the newest version by creation date",
			strategy: types.VersioningStrategyChronological,
			versions: semverOnly,
			want:     []string{"on-2.0.0", "on-2.1.0"},
		},
		{
			name:     "legacy orders by SemVer while every name parses",
			strategy: types.VersioningStrategyLegacy,
			versions: semverOnly,
			want:     []string{"on-2.0.0"},
		},
		{
			name:     "legacy orders by creation date once any name does not parse",
			strategy: types.VersioningStrategyLegacy,
			versions: slices.Concat(semverOnly, []types.ApplicationVersion{nightly}),
			want:     []string{"on-2.0.0", "on-2.1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			application := types.Application{
				VersioningStrategy: tt.strategy,
				Versions:           tt.versions,
			}
			g.Expect(targetNames(deploymentsBehind(application, v205, deployments))).To(Equal(tt.want))
		})
	}
}

func targetNames(deployments []types.DeploymentPendingUpdate) []string {
	result := make([]string, len(deployments))
	for i, deployment := range deployments {
		result[i] = deployment.DeploymentTargetName
	}
	return result
}

func TestVisibleDeployments(t *testing.T) {
	customerA := uuid.New()
	customerB := uuid.New()
	partner := uuid.New()

	internal := types.DeploymentPendingUpdate{DeploymentTargetName: "internal"}
	ofCustomerA := types.DeploymentPendingUpdate{
		DeploymentTargetName:   "customer-a",
		CustomerOrganizationID: &customerA,
	}
	ofCustomerB := types.DeploymentPendingUpdate{
		DeploymentTargetName:   "customer-b",
		CustomerOrganizationID: &customerB,
		PartnerOrganizationID:  &partner,
	}
	all := []types.DeploymentPendingUpdate{internal, ofCustomerA, ofCustomerB}

	tests := []struct {
		name     string
		audience audience
		want     []string
	}{
		{
			name:     "the vendor's team sees every affected deployment",
			audience: audience{},
			want:     []string{"internal", "customer-a", "customer-b"},
		},
		{
			name:     "a customer sees only its own deployments",
			audience: audience{customerOrganizationID: &customerA},
			want:     []string{"customer-a"},
		},
		{
			name:     "a partner sees the deployments of the customers it manages",
			audience: audience{partnerOrganizationID: &partner},
			want:     []string{"customer-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(targetNames(visibleDeployments(all, tt.audience))).To(Equal(tt.want))
		})
	}
}

func TestRecipientsByAudience(t *testing.T) {
	g := NewWithT(t)
	customerA := uuid.New()
	partner := uuid.New()
	vendor1 := types.NotificationRecipient{ID: uuid.New()}
	customer1 := types.NotificationRecipient{ID: uuid.New(), CustomerOrganizationID: new(customerA)}
	vendor2 := types.NotificationRecipient{ID: uuid.New()}
	partner1 := types.NotificationRecipient{ID: uuid.New(), PartnerOrganizationID: &partner}
	customer2 := types.NotificationRecipient{ID: uuid.New(), CustomerOrganizationID: new(customerA)}

	groups := recipientsByAudience([]types.NotificationRecipient{vendor1, customer1, vendor2, partner1, customer2})

	g.Expect(groups).To(Equal([]audienceRecipients{
		{audience: audience{}, recipients: []types.NotificationRecipient{vendor1, vendor2}},
		{
			audience:   audience{customerOrganizationID: customer1.CustomerOrganizationID},
			recipients: []types.NotificationRecipient{customer1, customer2},
		},
		{audience: audience{partnerOrganizationID: &partner}, recipients: []types.NotificationRecipient{partner1}},
	}))
}
