package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type SupportBundleStatus string

const (
	SupportBundleStatusInitialized SupportBundleStatus = "initialized"
	SupportBundleStatusCreated     SupportBundleStatus = "created"
	SupportBundleStatusResolved    SupportBundleStatus = "resolved"
	SupportBundleStatusCanceled    SupportBundleStatus = "canceled"
)

type SupportBundleConfigurationEnvVar struct {
	OrganizationID uuid.UUID `db:"organization_id"`
	Name           string    `db:"name"`
	Redacted       bool      `db:"redacted"`
}

type SupportBundleConfigurationScript struct {
	ID             uuid.UUID      `db:"id"`
	CreatedAt      time.Time      `db:"created_at"`
	OrganizationID uuid.UUID      `db:"organization_id"`
	DeploymentType DeploymentType `db:"deployment_type"`
	Name           string         `db:"name"`
	Description    *string        `db:"description"`
	Content        string         `db:"content"`
	Enabled        bool           `db:"enabled"`
}

type SupportBundle struct {
	ID                           uuid.UUID           `db:"id"`
	CreatedAt                    time.Time           `db:"created_at"`
	OrganizationID               uuid.UUID           `db:"organization_id"`
	CustomerOrganizationID       uuid.UUID           `db:"customer_organization_id"`
	CreatedByUserAccountID       *uuid.UUID          `db:"created_by_user_account_id"`
	DeploymentType               DeploymentType      `db:"deployment_type"`
	Title                        string              `db:"title"`
	Description                  *string             `db:"description"`
	Status                       SupportBundleStatus `db:"status"`
	BundleSecret                 dbcrypto.String     `db:"bundle_secret"`
	BundleSecretExpiresAt        *time.Time          `db:"bundle_secret_expires_at"`
	StatusChangedByUserAccountID *uuid.UUID          `db:"status_changed_by_user_account_id"`
	StatusChangedAt              *time.Time          `db:"status_changed_at"`
}

type SupportBundleWithDetails struct {
	SupportBundle
	CreatedByUserName        *string    `db:"created_by_user_name"`
	CreatedByImageID         *uuid.UUID `db:"created_by_image_id"`
	CustomerOrganizationName string     `db:"customer_organization_name"`
	ResourceCount            int64      `db:"resource_count"`
	CommentCount             int64      `db:"comment_count"`
	LastCommentAt            *time.Time `db:"last_comment_at"`
	StatusChangedByUserName  *string    `db:"status_changed_by_user_name"`
	StatusChangedByImageID   *uuid.UUID `db:"status_changed_by_image_id"`
}

type SupportBundleResourceContentType string

const (
	SupportBundleResourceContentTypeText SupportBundleResourceContentType = "text/plain"
	SupportBundleResourceContentTypeLog  SupportBundleResourceContentType = "text/x-log"
	SupportBundleResourceContentTypeYAML SupportBundleResourceContentType = "application/yaml"
	SupportBundleResourceContentTypeJSON SupportBundleResourceContentType = "application/json"
)

var ErrInvalidSupportBundleResourceContentType = errors.New("invalid support bundle resource content type")

func ParseSupportBundleResourceContentType(value string) (SupportBundleResourceContentType, error) {
	switch SupportBundleResourceContentType(value) {
	case SupportBundleResourceContentTypeText:
		return SupportBundleResourceContentTypeText, nil
	case SupportBundleResourceContentTypeLog:
		return SupportBundleResourceContentTypeLog, nil
	case SupportBundleResourceContentTypeYAML:
		return SupportBundleResourceContentTypeYAML, nil
	case SupportBundleResourceContentTypeJSON:
		return SupportBundleResourceContentTypeJSON, nil
	default:
		return "", fmt.Errorf("%w: %v", ErrInvalidSupportBundleResourceContentType, value)
	}
}

func (t SupportBundleResourceContentType) FileExtension() string {
	switch t {
	case SupportBundleResourceContentTypeLog:
		return ".log"
	case SupportBundleResourceContentTypeYAML:
		return ".yaml"
	case SupportBundleResourceContentTypeJSON:
		return ".json"
	default:
		return ".txt"
	}
}

type SupportBundleResourceSummary struct {
	ID              uuid.UUID                        `db:"id"`
	CreatedAt       time.Time                        `db:"created_at"`
	SupportBundleID uuid.UUID                        `db:"support_bundle_id"`
	Name            string                           `db:"name"`
	ContentType     SupportBundleResourceContentType `db:"content_type"`
	// Size is the length of the content in bytes. It is nil for resources uploaded before it was
	// recorded, because the content is stored compressed and encrypted and cannot be measured in SQL.
	Size *int64 `db:"size"`
}

type SupportBundleResource struct {
	SupportBundleResourceSummary
	// The content is a file the collect script gathered verbatim, and is served back as a download
	// and rendered in a <pre>, so it must not be trimmed if it ever reaches a JSON request body.
	Content dbcrypto.String `db:"content" trim:"-"`
}

type SupportBundleComment struct {
	ID              uuid.UUID  `db:"id"`
	CreatedAt       time.Time  `db:"created_at"`
	SupportBundleID uuid.UUID  `db:"support_bundle_id"`
	UserAccountID   *uuid.UUID `db:"user_account_id"`
	Content         string     `db:"content"`
}

type SupportBundleCommentWithUser struct {
	SupportBundleComment
	UserName    *string    `db:"user_name"`
	UserImageID *uuid.UUID `db:"user_image_id"`
}
