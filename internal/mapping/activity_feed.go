package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// ActivityFeedEntryToAPI returns a mapper that converts an activity feed entry to its API representation,
// showing its user by the same rule as deploymentRevisionCreatorToAPI.
func ActivityFeedEntryToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(activityfeed.Entry) api.ActivityFeedEntry {
	userToAPI := activityFeedUserToAPI(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(e activityfeed.Entry) api.ActivityFeedEntry {
		return api.ActivityFeedEntry{
			ID:                       e.ID,
			CreatedAt:                e.CreatedAt,
			Type:                     e.Type,
			ResourceID:               e.ResourceID,
			ResourceName:             e.ResourceName,
			Subject:                  e.Subject,
			Detail:                   e.Detail,
			User:                     userToAPI(e.User),
			CustomerOrganizationID:   e.CustomerOrganizationID,
			CustomerOrganizationName: e.CustomerOrganizationName,
			PartnerOrganizationID:    e.PartnerOrganizationID,
			PartnerOrganizationName:  e.PartnerOrganizationName,
		}
	}
}

func activityFeedUserToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(*types.ActivityUser) *api.ActivityFeedUser {
	viewerRank := organizationKindRank(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(u *types.ActivityUser) *api.ActivityFeedUser {
		if u == nil {
			return nil
		}
		showIdentity := organizationKindRank(u.CustomerOrganizationID, u.PartnerOrganizationID) >= viewerRank
		if u.Deleted && !showIdentity {
			return nil
		}
		user := &api.ActivityFeedUser{
			CustomerOrganizationID: u.CustomerOrganizationID,
			PartnerOrganizationID:  u.PartnerOrganizationID,
			Deleted:                u.Deleted,
		}
		if showIdentity {
			user.Name = u.Name
			user.ImageURL = CreateImageURL(u.ImageID)
		}
		return user
	}
}
