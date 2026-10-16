package types

import "github.com/google/uuid"

type ApplicationVersionResource struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	ApplicationVersionID uuid.UUID `db:"application_version_id" json:"applicationVersionId"`
	Name                 string    `db:"name" json:"name"`
	Content              string    `db:"content" json:"content"`
	VisibleToCustomers   bool      `db:"visible_to_customers" json:"visibleToCustomers"`
}
