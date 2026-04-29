package types

import (
	"time"

	"github.com/google/uuid"
)

type OpenTofuState struct {
	ID             uuid.UUID  `db:"id"`
	DeploymentID   uuid.UUID  `db:"deployment_id"`
	OrganizationID uuid.UUID  `db:"organization_id"`
	S3Key          string     `db:"s3_key"`
	LockID         *string    `db:"lock_id"`
	LockInfo       *string    `db:"lock_info"`
	LockedAt       *time.Time `db:"locked_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
	CreatedAt      time.Time  `db:"created_at"`
}
