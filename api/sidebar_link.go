package api

import (
	"time"

	"github.com/google/uuid"
)

type SidebarLink struct {
	ID                     uuid.UUID  `json:"id"`
	CreatedAt              time.Time  `json:"createdAt"`
	OrganizationID         uuid.UUID  `json:"organizationId"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
	Name                   string     `json:"name"`
	Link                   string     `json:"link"`
}

type CreateUpdateSidebarLinkRequest struct {
	Name string `json:"name"`
	Link string `json:"link"`
}
