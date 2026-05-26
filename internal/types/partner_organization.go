package types

import (
	"time"

	"github.com/google/uuid"
)

type PartnerOrganization struct {
	ID             uuid.UUID `db:"id"              json:"id"`
	CreatedAt      time.Time `db:"created_at"      json:"createdAt"`
	OrganizationID uuid.UUID `db:"organization_id" json:"organizationId"`
	Name           string    `db:"name"            json:"name"`
}

type PartnerOrganizationWithUsage struct {
	PartnerOrganization
	UserCount                 int64 `db:"user_count"                   json:"userCount"`
	CustomerOrganizationCount int64 `db:"customer_organization_count"  json:"customerOrganizationCount"`
}
