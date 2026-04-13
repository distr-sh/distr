package types

import (
	"time"

	"github.com/google/uuid"
)

type CustomerOrganizationLink struct {
	ID                     uuid.UUID `db:"id" json:"id"`
	CreatedAt              time.Time `db:"created_at" json:"createdAt"`
	CustomerOrganizationID uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId"`
	Name                   string    `db:"name" json:"name"`
	Link                   string    `db:"link" json:"link"`
}
