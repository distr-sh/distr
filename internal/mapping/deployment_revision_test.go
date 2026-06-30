package mapping_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestDeploymentRevisionToAPI_CreatorVisibility(t *testing.T) {
	customerOrg := uuid.New()
	partnerOrg := uuid.New()
	creatorID := uuid.New()
	creatorImageID := uuid.New()
	creatorName := "Jane Doe"
	creatorEmail := "jane@acme.com"

	const (
		vendor = iota
		partner
		customer
	)

	creator := func(kind int, deleted bool) types.DeploymentRevisionWithCreator {
		r := types.DeploymentRevisionWithCreator{
			ID:               uuid.New(),
			CreatedByID:      &creatorID,
			CreatedByName:    &creatorName,
			CreatedByEmail:   &creatorEmail,
			CreatedByImageID: &creatorImageID,
			CreatedByDeleted: deleted,
		}
		switch kind {
		case partner:
			r.CreatedByPartnerOrganizationID = &partnerOrg
		case customer:
			r.CreatedByCustomerOrganizationID = &customerOrg
		}
		return r
	}

	tests := []struct {
		name            string
		viewerCustomer  *uuid.UUID
		viewerPartner   *uuid.UUID
		input           types.DeploymentRevisionWithCreator
		wantNil         bool
		wantIdentity    bool
		wantCustomerOrg *uuid.UUID
		wantPartnerOrg  *uuid.UUID
		wantDeleted     bool
	}{
		{
			name:         "vendor viewer sees vendor creator",
			input:        creator(vendor, false),
			wantIdentity: true,
		},
		{
			name:           "vendor viewer sees partner creator",
			input:          creator(partner, false),
			wantIdentity:   true,
			wantPartnerOrg: &partnerOrg,
		},
		{
			name:            "vendor viewer sees customer creator",
			input:           creator(customer, false),
			wantIdentity:    true,
			wantCustomerOrg: &customerOrg,
		},

		{
			name:          "partner viewer sees empty creator for vendor creator",
			viewerPartner: &partnerOrg,
			input:         creator(vendor, false),
			wantIdentity:  false,
		},
		{
			name:           "partner viewer sees partner creator",
			viewerPartner:  &partnerOrg,
			input:          creator(partner, false),
			wantIdentity:   true,
			wantPartnerOrg: &partnerOrg,
		},
		{
			name:            "partner viewer sees customer creator",
			viewerPartner:   &partnerOrg,
			input:           creator(customer, false),
			wantIdentity:    true,
			wantCustomerOrg: &customerOrg,
		},

		{
			name:           "customer viewer sees empty creator for vendor creator",
			viewerCustomer: &customerOrg,
			input:          creator(vendor, false),
			wantIdentity:   false,
		},
		{
			name:           "customer viewer shows partner org without identity",
			viewerCustomer: &customerOrg,
			input:          creator(partner, false),
			wantIdentity:   false,
			wantPartnerOrg: &partnerOrg,
		},
		{
			name:            "customer viewer sees customer creator",
			viewerCustomer:  &customerOrg,
			input:           creator(customer, false),
			wantIdentity:    true,
			wantCustomerOrg: &customerOrg,
		},

		{
			name:         "vendor viewer sees deleted creator",
			input:        creator(vendor, true),
			wantIdentity: true,
			wantDeleted:  true,
		},
		{
			name:          "partner viewer hides deleted creator",
			viewerPartner: &partnerOrg,
			input:         creator(vendor, true),
			wantNil:       true,
		},
		{
			name:           "customer viewer hides deleted creator",
			viewerCustomer: &customerOrg,
			input:          creator(vendor, true),
			wantNil:        true,
		},

		{
			name:    "no creator yields nil",
			input:   types.DeploymentRevisionWithCreator{ID: uuid.New()},
			wantNil: true,
		},
	}

	expectUUIDPtr := func(g Gomega, actual, expected *uuid.UUID) {
		if expected == nil {
			g.Expect(actual).To(BeNil())
		} else {
			g.Expect(actual).To(Equal(expected))
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := mapping.DeploymentRevisionToAPI(tt.viewerCustomer, tt.viewerPartner)(tt.input)
			g.Expect(result.ID).To(Equal(tt.input.ID))

			if tt.wantNil {
				g.Expect(result.CreatedBy).To(BeNil())
				return
			}

			g.Expect(result.CreatedBy).NotTo(BeNil())
			if tt.wantIdentity {
				expectUUIDPtr(g, result.CreatedBy.ID, &creatorID)
				expectUUIDPtr(g, result.CreatedBy.ImageID, &creatorImageID)
				g.Expect(result.CreatedBy.Name).To(Equal(creatorName))
				g.Expect(result.CreatedBy.Email).To(Equal(creatorEmail))
			} else {
				g.Expect(result.CreatedBy.ID).To(BeNil())
				g.Expect(result.CreatedBy.ImageID).To(BeNil())
				g.Expect(result.CreatedBy.Name).To(BeEmpty())
				g.Expect(result.CreatedBy.Email).To(BeEmpty())
			}
			expectUUIDPtr(g, result.CreatedBy.CustomerOrganizationID, tt.wantCustomerOrg)
			expectUUIDPtr(g, result.CreatedBy.PartnerOrganizationID, tt.wantPartnerOrg)
			g.Expect(result.CreatedBy.Deleted).To(Equal(tt.wantDeleted))
		})
	}
}
