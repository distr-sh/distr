package types

import "github.com/google/uuid"

type ApplicationVersionResource struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	ApplicationVersionID uuid.UUID `db:"application_version_id" json:"applicationVersionId"`
	Name                 string    `db:"name" json:"name"`
	// The content is rendered as markdown, where leading whitespace is syntax.
	Content            string `db:"content" json:"content" trim:"-"`
	VisibleToCustomers bool   `db:"visible_to_customers" json:"visibleToCustomers"`
}
