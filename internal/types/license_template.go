package types

import (
	"time"

	"github.com/google/uuid"
)

type LicenseTemplate struct {
	ID                        uuid.UUID `db:"id"                           json:"id"`
	CreatedAt                 time.Time `db:"created_at"                   json:"createdAt"`
	Name                      string    `db:"name"                         json:"name"`
	OrganizationID            uuid.UUID `db:"organization_id"              json:"-"`
	PayloadTemplate           string    `db:"payload_template"             json:"payloadTemplate"`
	ExpirationGracePeriodDays int       `db:"expiration_grace_period_days" json:"expirationGracePeriodDays"`
}
