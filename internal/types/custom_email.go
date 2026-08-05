package types

import (
	"time"

	"github.com/google/uuid"
)

// Overrides the instance mailer. There is at most one per organization.
type CustomEmailConfiguration struct {
	ID                     uuid.UUID  `db:"id"                         json:"id"`
	CreatedAt              time.Time  `db:"created_at"                 json:"createdAt"`
	UpdatedAt              time.Time  `db:"updated_at"                 json:"updatedAt"`
	UpdatedByUserAccountID *uuid.UUID `db:"updated_by_user_account_id" json:"updatedByUserAccountId"`
	OrganizationID         uuid.UUID  `db:"organization_id"            json:"organizationId"`
	// A disabled configuration is kept but ignored, so the instance mailer is used instead.
	Enabled         bool   `db:"enabled"           json:"enabled"`
	FromAddress     string `db:"from_address"      json:"fromAddress"`
	SMTPHost        string `db:"smtp_host"         json:"smtpHost"`
	SMTPPort        int    `db:"smtp_port"         json:"smtpPort"`
	SMTPUsername    string `db:"smtp_username"     json:"smtpUsername"`
	SMTPPassword    string `db:"smtp_password"     json:"-"`
	SMTPImplicitTLS bool   `db:"smtp_implicit_tls" json:"smtpImplicitTls"`
}
