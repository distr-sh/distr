package api

import (
	"net/mail"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

// CustomEmailConfiguration is the API representation of an organization's email configuration.
// The SMTP password is write-only and therefore only reported as being set or not.
type CustomEmailConfiguration struct {
	ID              uuid.UUID `json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	OrganizationID  uuid.UUID `json:"organizationId"`
	Enabled         bool      `json:"enabled"`
	FromAddress     string    `json:"fromAddress"`
	SMTPHost        string    `json:"smtpHost"`
	SMTPPort        int       `json:"smtpPort"`
	SMTPUsername    string    `json:"smtpUsername"`
	SMTPPasswordSet bool      `json:"smtpPasswordSet"`
	SMTPImplicitTLS bool      `json:"smtpImplicitTls"`
}

// CustomEmailSettings are the mail transport settings shared by the update and the test request.
type CustomEmailSettings struct {
	FromAddress  string `json:"fromAddress"`
	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUsername string `json:"smtpUsername"`
	// SMTPPassword is omitted to keep the currently stored password, and set to an empty string
	// to clear it.
	SMTPPassword    *string `json:"smtpPassword"`
	SMTPImplicitTLS bool    `json:"smtpImplicitTls"`
}

// Normalize reduces the SMTP host to the bare hostname it is validated and stored as, so that a
// value pasted as a URL is accepted as well.
func (r *CustomEmailSettings) Normalize() {
	r.FromAddress = strings.TrimSpace(r.FromAddress)
	r.SMTPHost = validation.NormalizeHostname(r.SMTPHost)
	r.SMTPUsername = strings.TrimSpace(r.SMTPUsername)
}

func (r *CustomEmailSettings) Validate() error {
	if _, err := mail.ParseAddress(r.FromAddress); err != nil {
		return validation.NewValidationFailedError("fromAddress must be a valid email address")
	}
	if err := validation.ValidateHostname(r.SMTPHost); err != nil {
		return err
	}
	if r.SMTPPort < 1 || r.SMTPPort > 65535 {
		return validation.NewValidationFailedError("smtpPort must be between 1 and 65535")
	}
	return nil
}

type UpdateCustomEmailConfigurationRequest struct {
	CustomEmailSettings
	// Enabled reports whether mails should actually be sent through this configuration. A
	// disabled configuration is kept but ignored, so the instance mailer is used instead.
	Enabled bool `json:"enabled"`
}

type TestCustomEmailConfigurationRequest struct {
	CustomEmailSettings
}
