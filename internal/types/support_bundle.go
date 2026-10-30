package types

import (
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/contenttype"
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

type SupportBundleKubernetesConfiguration struct {
	OrganizationID       uuid.UUID `db:"organization_id"`
	IncludeConfigMapData bool      `db:"include_configmap_data"`
	IncludeSecretData    bool      `db:"include_secret_data"`
}

// DefaultSupportBundleKubernetesConfiguration applies to an organization that never saved one.
func DefaultSupportBundleKubernetesConfiguration(orgID uuid.UUID) SupportBundleKubernetesConfiguration {
	return SupportBundleKubernetesConfiguration{OrganizationID: orgID, IncludeConfigMapData: true}
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

var SupportBundleResourceContentTypes = []string{
	contenttype.MediaTypeTextPlain,
	contenttype.MediaTypeTextXLog,
	contenttype.MediaTypeYAML,
	contenttype.MediaTypeJSON,
}

func ParseSupportBundleResourceContentType(value string) string {
	for _, contentType := range SupportBundleResourceContentTypes {
		if value == contentType {
			return value
		}
	}
	return contenttype.MediaTypeTextPlain
}

func SupportBundleResourceFileExtension(contentType string) string {
	switch contentType {
	case contenttype.MediaTypeTextXLog:
		return ".log"
	case contenttype.MediaTypeYAML:
		return ".yaml"
	case contenttype.MediaTypeJSON:
		return ".json"
	default:
		return ".txt"
	}
}

// SupportBundleResourceContentTypeOfFile is the content type whose SupportBundleResourceFileExtension
// the name ends with, or plain text.
func SupportBundleResourceContentTypeOfFile(name string) string {
	for _, contentType := range []string{
		contenttype.MediaTypeTextXLog,
		contenttype.MediaTypeYAML,
		contenttype.MediaTypeJSON,
	} {
		if strings.HasSuffix(name, SupportBundleResourceFileExtension(contentType)) {
			return contentType
		}
	}
	return contenttype.MediaTypeTextPlain
}

type SupportBundleResourceSummary struct {
	ID              uuid.UUID `db:"id"`
	CreatedAt       time.Time `db:"created_at"`
	SupportBundleID uuid.UUID `db:"support_bundle_id"`
	Name            string    `db:"name"`
	ContentType     string    `db:"content_type"`
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
