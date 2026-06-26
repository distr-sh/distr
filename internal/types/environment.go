package types

import (
	"time"

	"github.com/google/uuid"
)

type Environment struct {
	ID                  uuid.UUID  `db:"id" json:"id"`
	CreatedAt           time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updatedAt"`
	OrganizationID      uuid.UUID  `db:"organization_id" json:"organizationId"`
	Name                string     `db:"name" json:"name"`
	Description         string     `db:"description" json:"description"`
	SortOrder           int        `db:"sort_order" json:"sortOrder"`
	IsProduction        bool       `db:"is_production" json:"isProduction"`
	AllowDynamicTargets bool       `db:"allow_dynamic_targets" json:"allowDynamicTargets"`
	RetentionPolicyID   *uuid.UUID `db:"retention_policy_id" json:"retentionPolicyId,omitempty"`
}
