package mapping_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestSupportBundleToAPI_UserVisibility(t *testing.T) {
	customerOrg := uuid.New()
	partnerOrg := uuid.New()
	userName := "Jane Doe"

	type membership struct {
		customerOrgID *uuid.UUID
		partnerOrgID  *uuid.UUID
		deleted       bool
	}
	vendor := membership{}
	partner := membership{partnerOrgID: &partnerOrg}
	customer := membership{customerOrgID: &customerOrg}
	deleted := membership{deleted: true}

	// Every actor but the one under test is deleted, so that checking the wrong actor's membership hides the name
	// from a customer or partner viewer who should see it.
	createdBy := func(m membership) types.SupportBundleWithDetails {
		return types.SupportBundleWithDetails{
			CreatedByUserName:               &userName,
			CreatedByCustomerOrganizationID: m.customerOrgID,
			CreatedByPartnerOrganizationID:  m.partnerOrgID,
			CreatedByDeleted:                m.deleted,
			StatusChangedByUserName:         &userName,
			StatusChangedByDeleted:          true,
		}
	}
	statusChangedBy := func(m membership) types.SupportBundleWithDetails {
		return types.SupportBundleWithDetails{
			CreatedByUserName:                     &userName,
			CreatedByDeleted:                      true,
			StatusChangedByUserName:               &userName,
			StatusChangedByCustomerOrganizationID: m.customerOrgID,
			StatusChangedByPartnerOrganizationID:  m.partnerOrgID,
			StatusChangedByDeleted:                m.deleted,
		}
	}
	commentedBy := func(m membership) types.SupportBundleCommentWithUser {
		return types.SupportBundleCommentWithUser{
			UserName:                   &userName,
			UserCustomerOrganizationID: m.customerOrgID,
			UserPartnerOrganizationID:  m.partnerOrgID,
			UserDeleted:                m.deleted,
		}
	}

	tests := []struct {
		name           string
		viewerCustomer *uuid.UUID
		viewerPartner  *uuid.UUID
		user           membership
		wantName       bool
	}{
		{name: "vendor viewer sees deleted user", user: deleted, wantName: true},
		{name: "partner viewer does not see vendor user", viewerPartner: &partnerOrg, user: vendor},
		{name: "partner viewer sees partner user", viewerPartner: &partnerOrg, user: partner, wantName: true},
		{name: "partner viewer sees customer user", viewerPartner: &partnerOrg, user: customer, wantName: true},
		{name: "partner viewer does not see deleted user", viewerPartner: &partnerOrg, user: deleted},
		{name: "customer viewer does not see vendor user", viewerCustomer: &customerOrg, user: vendor},
		{name: "customer viewer does not see partner user", viewerCustomer: &customerOrg, user: partner},
		{name: "customer viewer sees customer user", viewerCustomer: &customerOrg, user: customer, wantName: true},
		{name: "customer viewer does not see deleted user", viewerCustomer: &customerOrg, user: deleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			expectName := func(actual *string) {
				if tt.wantName {
					g.Expect(actual).To(HaveValue(Equal(userName)))
				} else {
					g.Expect(actual).To(BeNil())
				}
			}
			bundleToAPI := mapping.SupportBundleToAPI(tt.viewerCustomer, tt.viewerPartner)
			commentToAPI := mapping.SupportBundleCommentToAPI(tt.viewerCustomer, tt.viewerPartner)

			expectName(bundleToAPI(createdBy(tt.user)).CreatedByUserName)
			expectName(bundleToAPI(statusChangedBy(tt.user)).StatusChangedByUserName)
			expectName(commentToAPI(commentedBy(tt.user)).UserName)
		})
	}
}
