package mapping

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

//nolint:staticcheck // the deprecated field is what this test is about
func TestCreateDeploymentTargetToInternalCustomerOrganization(t *testing.T) {
	g := NewWithT(t)
	flat, nested := uuid.New(), uuid.New()

	dt := CreateDeploymentTargetToInternal(api.CreateDeploymentTargetRequest{
		CustomerOrganization: &api.EntityReference{ID: nested},
	})
	g.Expect(dt.CustomerOrganizationID).To(HaveValue(Equal(nested)))

	dt = CreateDeploymentTargetToInternal(api.CreateDeploymentTargetRequest{
		CustomerOrganizationID: &flat,
		CustomerOrganization:   &api.EntityReference{ID: nested},
	})
	g.Expect(dt.CustomerOrganizationID).To(HaveValue(Equal(flat)))

	dt = CreateDeploymentTargetToInternal(api.CreateDeploymentTargetRequest{
		CustomerOrganization: &api.EntityReference{ID: uuid.Nil},
	})
	g.Expect(dt.CustomerOrganizationID).To(BeNil())
}
