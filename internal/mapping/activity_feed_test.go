package mapping_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestActivityFeedEntryToAPI_UserVisibility(t *testing.T) {
	customerOrg := uuid.New()
	partnerOrg := uuid.New()
	userName := "Jane Doe"
	imageID := uuid.New()
	imageURL := mapping.CreateImageURL(&imageID)

	vendor := types.ActivityUser{Name: &userName, ImageID: &imageID}
	partner := types.ActivityUser{Name: &userName, ImageID: &imageID, PartnerOrganizationID: &partnerOrg}
	customer := types.ActivityUser{Name: &userName, ImageID: &imageID, CustomerOrganizationID: &customerOrg}
	deleted := types.ActivityUser{Name: &userName, ImageID: &imageID, Deleted: true}

	tests := []struct {
		name           string
		viewerCustomer *uuid.UUID
		viewerPartner  *uuid.UUID
		user           types.ActivityUser
		wantUser       bool
		wantIdentity   bool
	}{
		{name: "vendor viewer sees deleted user", user: deleted, wantUser: true, wantIdentity: true},
		{
			name:          "partner viewer knows vendor user by organization",
			viewerPartner: &partnerOrg,
			user:          vendor,
			wantUser:      true,
		},
		{
			name:          "partner viewer sees partner user",
			viewerPartner: &partnerOrg,
			user:          partner,
			wantUser:      true,
			wantIdentity:  true,
		},
		{name: "partner viewer does not see deleted user", viewerPartner: &partnerOrg, user: deleted},
		{
			name:           "customer viewer knows partner user by organization",
			viewerCustomer: &customerOrg,
			user:           partner,
			wantUser:       true,
		},
		{
			name:           "customer viewer sees customer user",
			viewerCustomer: &customerOrg,
			user:           customer,
			wantUser:       true,
			wantIdentity:   true,
		},
		{name: "customer viewer does not see deleted user", viewerCustomer: &customerOrg, user: deleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			toAPI := mapping.ActivityFeedEntryToAPI(tt.viewerCustomer, tt.viewerPartner)
			result := toAPI(activityfeed.Entry{User: &tt.user})
			if !tt.wantUser {
				g.Expect(result.User).To(BeNil())
				return
			}
			g.Expect(result.User).NotTo(BeNil())
			g.Expect(result.User.CustomerOrganizationID).To(Equal(tt.user.CustomerOrganizationID))
			g.Expect(result.User.PartnerOrganizationID).To(Equal(tt.user.PartnerOrganizationID))
			g.Expect(result.User.Deleted).To(Equal(tt.user.Deleted))
			if tt.wantIdentity {
				g.Expect(result.User.Name).To(HaveValue(Equal(userName)))
				g.Expect(result.User.ImageURL).To(Equal(imageURL))
			} else {
				g.Expect(result.User.Name).To(BeNil())
				g.Expect(result.User.ImageURL).To(BeNil())
			}
		})
	}
}
