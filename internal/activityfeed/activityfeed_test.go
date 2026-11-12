package activityfeed_test

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestLatest_partnerNeverReceivesActivityOfCustomersItDoesNotManage(t *testing.T) {
	g := NewWithT(t)
	now := time.Now()
	managed := types.CustomerOrganization{ID: uuid.New(), Name: "Managed"}
	otherPartner := uuid.New()
	vendorOwned := uuid.New()

	entries := make([]activityfeed.Entry, 0, 2+2*activityfeed.Limit)
	entries = append(entries,
		activityfeed.Entry{ID: uuid.New(), CreatedAt: now.Add(-time.Hour), CustomerOrganizationID: &managed.ID},
		activityfeed.Entry{ID: uuid.New(), CreatedAt: now.Add(-2 * time.Hour)},
	)
	for i := range activityfeed.Limit {
		createdAt := now.Add(-time.Duration(i))
		entries = append(entries,
			activityfeed.Entry{ID: uuid.New(), CreatedAt: createdAt, CustomerOrganizationID: &otherPartner},
			activityfeed.Entry{ID: uuid.New(), CreatedAt: createdAt, CustomerOrganizationID: &vendorOwned},
		)
	}

	result := activityfeed.Latest(entries, []types.CustomerOrganization{managed})

	g.Expect(result).To(HaveExactElements(
		SatisfyAll(
			HaveField("ID", entries[0].ID),
			HaveField("CustomerOrganizationName", HaveValue(Equal("Managed"))),
		),
		SatisfyAll(
			HaveField("ID", entries[1].ID),
			HaveField("CustomerOrganizationName", BeNil()),
		),
	))
}
