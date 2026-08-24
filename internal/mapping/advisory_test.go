package mapping_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestAdvisoryToAPI_CreatorVisibility(t *testing.T) {
	customerOrg := uuid.New()
	partnerOrg := uuid.New()
	imageID := uuid.New()
	name := "Jane Doe"

	advisory := types.AdvisoryWithDetails{
		Advisory:          types.Advisory{ID: uuid.New(), Title: "CVE-2026-0001 in the gateway"},
		CreatedByUserName: &name,
		CreatedByImageID:  &imageID,
	}

	tests := []struct {
		name           string
		viewerCustomer *uuid.UUID
		viewerPartner  *uuid.UUID
		wantCreator    bool
	}{
		{name: "vendor", wantCreator: true},
		{name: "partner", viewerPartner: &partnerOrg},
		{name: "customer", viewerCustomer: &customerOrg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := mapping.AdvisoryToAPI(tt.viewerCustomer, tt.viewerPartner)(advisory)
			if tt.wantCreator {
				g.Expect(result.CreatedByUserName).To(HaveValue(Equal(name)))
				g.Expect(result.CreatedByImageURL).NotTo(BeNil())
			} else {
				g.Expect(result.CreatedByUserName).To(BeNil())
				g.Expect(result.CreatedByImageURL).To(BeNil())
			}
			g.Expect(result.Title).To(Equal(advisory.Title))
		})
	}
}
