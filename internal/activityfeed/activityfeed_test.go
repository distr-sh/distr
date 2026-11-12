package activityfeed_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestRestrictToCustomerOrganizations_partnerNeverReceivesActivityOfCustomersItDoesNotManage(t *testing.T) {
	g := NewWithT(t)
	managed := types.CustomerOrganization{ID: uuid.New(), Name: "Managed"}
	otherPartner := uuid.New()
	vendorOwned := uuid.New()
	entries := []activityfeed.Entry{
		{ID: uuid.New(), CustomerOrganizationID: &otherPartner},
		{ID: uuid.New(), CustomerOrganizationID: &managed.ID},
		{ID: uuid.New(), CustomerOrganizationID: &vendorOwned},
		{ID: uuid.New()},
	}

	result := activityfeed.RestrictToCustomerOrganizations(entries, []types.CustomerOrganization{managed})

	g.Expect(result).To(HaveExactElements(
		SatisfyAll(
			HaveField("ID", entries[1].ID),
			HaveField("CustomerOrganizationName", HaveValue(Equal("Managed"))),
		),
		SatisfyAll(
			HaveField("ID", entries[3].ID),
			HaveField("CustomerOrganizationName", BeNil()),
		),
	))
}
