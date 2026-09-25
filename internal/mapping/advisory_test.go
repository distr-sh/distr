package mapping_test

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestAdvisoryToAPI_VendorFieldVisibility(t *testing.T) {
	customerOrg := uuid.New()
	partnerOrg := uuid.New()
	imageID := uuid.New()
	name := "Jane Doe"
	published := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	resolved := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	advisory := types.AdvisoryWithDetails{
		Advisory: types.Advisory{
			ID:          uuid.New(),
			Title:       "CVE-2026-0001 in the gateway",
			PublishedAt: &published,
			ResolvedAt:  &resolved,
		},
		CreatedByUserName: &name,
		CreatedByImageID:  &imageID,
	}

	tests := []struct {
		name             string
		viewerCustomer   *uuid.UUID
		viewerPartner    *uuid.UUID
		wantVendorFields bool
	}{
		{name: "vendor", wantVendorFields: true},
		{name: "partner", viewerPartner: &partnerOrg},
		{name: "customer", viewerCustomer: &customerOrg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := mapping.AdvisoryToAPI(tt.viewerCustomer, tt.viewerPartner)(advisory)
			if tt.wantVendorFields {
				g.Expect(result.CreatedByUserName).To(HaveValue(Equal(name)))
				g.Expect(result.CreatedByImageURL).NotTo(BeNil())
				g.Expect(result.ResolvedAt).To(HaveValue(Equal(resolved)))
			} else {
				g.Expect(result.CreatedByUserName).To(BeNil())
				g.Expect(result.CreatedByImageURL).To(BeNil())
				g.Expect(result.ResolvedAt).To(BeNil())
			}
			g.Expect(result.PublishedAt).To(HaveValue(Equal(published)))
			g.Expect(result.Title).To(Equal(advisory.Title))
		})
	}
}
