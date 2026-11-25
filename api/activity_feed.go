package api

import (
	"time"

	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/google/uuid"
)

type ActivityFeedEntry struct {
	// ID is the record the entry comes from, so it is only unique together with Type: the creation and the
	// status change of a support bundle share the bundle's ID.
	ID           uuid.UUID              `json:"id"`
	CreatedAt    time.Time              `json:"createdAt"`
	Type         activityfeed.EntryType `json:"type"`
	ResourceID   uuid.UUID              `json:"resourceId"`
	ResourceName string                 `json:"resourceName"`
	Subject      string                 `json:"subject"`
	Detail       *string                `json:"detail,omitempty"`
	User         *ActivityFeedUser      `json:"user,omitempty"`

	CustomerOrganizationID   *uuid.UUID `json:"customerOrganizationId,omitempty"`
	CustomerOrganizationName *string    `json:"customerOrganizationName,omitempty"`
	PartnerOrganizationID    *uuid.UUID `json:"partnerOrganizationId,omitempty"`
	PartnerOrganizationName  *string    `json:"partnerOrganizationName,omitempty"`
}

// ActivityFeedUser is who did what an entry is about. Name and ImageURL are missing for a user the viewer may
// not know, who is then known by their organization.
type ActivityFeedUser struct {
	Name                   *string    `json:"name,omitempty"`
	ImageURL               *string    `json:"imageUrl,omitempty"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
	PartnerOrganizationID  *uuid.UUID `json:"partnerOrganizationId,omitempty"`
	Deleted                bool       `json:"deleted,omitempty"`
}
