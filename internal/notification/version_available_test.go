package notification

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

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

	names := func(deployments []types.DeploymentPendingUpdate) []string {
		result := make([]string, len(deployments))
		for i, deployment := range deployments {
			result[i] = deployment.DeploymentTargetName
		}
		return result
	}

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
			g.Expect(names(visibleDeployments(all, tt.recipient))).To(Equal(tt.want))
		})
	}
}
