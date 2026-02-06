package types

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentStatusNotificationConfiguration struct {
	ID                     uuid.UUID   `db:"id" json:"id"`
	CreatedAt              time.Time   `db:"created_at" json:"createdAt"`
	OrganizationID         uuid.UUID   `db:"organization_id" json:"organizationId"`
	CustomerOrganizationID *uuid.UUID  `db:"customer_organization_id" json:"customerOrganizationId"`
	Name                   string      `db:"name" json:"name"`
	Enabled                bool        `db:"enabled" json:"enabled"`
	DeploymentTargetIDs    []uuid.UUID `db:"deployment_target_ids" json:"deploymentTargetIds"`
	UserAccountIDs         []uuid.UUID `db:"user_account_ids" json:"userAccountIds"`

	// UserAccounts is only populated from the database. It is never used by insert or update operations.
	UserAccounts []UserAccount `db:"user_accounts" json:"userAccounts"`

	// DeploymentTargets is only populated from the database. It is never used by insert or update operations.
	DeploymentTargets []DeploymentTarget `db:"deployment_targets" json:"deploymentTargets"`
}
