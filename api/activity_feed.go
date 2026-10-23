package api

import (
	"time"

	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/google/uuid"
)

type ActivityFeedEntry struct {
	ID           uuid.UUID              `json:"id"`
	CreatedAt    time.Time              `json:"createdAt"`
	Type         activityfeed.EntryType `json:"type"`
	ResourceID   uuid.UUID              `json:"resourceId"`
	Subject      string                 `json:"subject"`
	Detail       *string                `json:"detail,omitempty"`
	UserName     *string                `json:"userName,omitempty"`
	UserImageUrl *string                `json:"userImageUrl,omitempty"`
}
