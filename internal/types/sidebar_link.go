package types

import (
	"time"

	"github.com/google/uuid"
)

type SidebarLink struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	OrganizationID         uuid.UUID  `db:"organization_id" json:"organizationId"`
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
	Name                   string     `db:"name" json:"name"`
	Link                   string     `db:"link" json:"link"`
}
