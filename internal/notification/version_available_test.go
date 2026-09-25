package notification

import (
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

	deployments := []types.DeploymentPendingUpdate{
		{DeploymentTargetName: "on-2.0.0", CurrentVersionID: v200.ID},
		{DeploymentTargetName: "on-2.1.0", CurrentVersionID: v210.ID},
	}

	tests := []struct {
		strategy types.VersioningStrategy
		want     []string
	}{
		{strategy: types.VersioningStrategySemver, want: []string{"on-2.0.0"}},
		// Under the chronological strategy the bugfix is the newest version, so both are behind it.
		{strategy: types.VersioningStrategyChronological, want: []string{"on-2.0.0", "on-2.1.0"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			g := NewWithT(t)
			application := types.Application{
				VersioningStrategy: tt.strategy,
				Versions:           []types.ApplicationVersion{v200, v210, v205},
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

	internal := types.DeploymentPendingUpdate{DeploymentTargetName: "internal", Entitled: true}
	ofCustomerA := types.DeploymentPendingUpdate{
		DeploymentTargetName:   "customer-a",
		CustomerOrganizationID: &customerA,
		Entitled:               true,
	}
	ofCustomerB := types.DeploymentPendingUpdate{
		DeploymentTargetName:   "customer-b",
		CustomerOrganizationID: &customerB,
		PartnerOrganizationID:  &partner,
		Entitled:               true,
	}
	notEntitled := types.DeploymentPendingUpdate{
		DeploymentTargetName:   "customer-a-old",
		CustomerOrganizationID: &customerA,
		Entitled:               false,
	}
	all := []types.DeploymentPendingUpdate{internal, ofCustomerA, ofCustomerB, notEntitled}

	tests := []struct {
		name      string
		recipient types.NotificationRecipient
		want      []string
	}{
		{
			name:      "a vendor recipient sees every affected deployment",
			recipient: types.NotificationRecipient{ID: uuid.New()},
			want:      []string{"internal", "customer-a", "customer-b", "customer-a-old"},
		},
		{
			name: "a customer recipient sees only entitled deployments of their own organization",
			recipient: types.NotificationRecipient{
				ID:                     uuid.New(),
				CustomerOrganizationID: &customerA,
			},
			want: []string{"customer-a"},
		},
		{
			name: "a partner recipient sees the deployments of the customers they manage",
			recipient: types.NotificationRecipient{
				ID:                    uuid.New(),
				PartnerOrganizationID: &partner,
			},
			want: []string{"customer-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(targetNames(visibleDeployments(all, tt.recipient))).To(Equal(tt.want))
		})
	}
}
