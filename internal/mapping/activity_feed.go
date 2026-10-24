package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/activityfeed"
)

func ActivityFeedEntryToAPI(e activityfeed.Entry) api.ActivityFeedEntry {
	result := api.ActivityFeedEntry{
		ID:         e.ID,
		CreatedAt:  e.CreatedAt,
		Type:       e.Type,
		ResourceID: e.ResourceID,
		Subject:    e.Subject,
		Detail:     e.Detail,
	}
	if e.User != nil {
		result.UserName = &e.User.Name
		result.UserImageUrl = CreateImageURL(e.User.ImageID)
	}
	return result
}
