package types

import (
	"time"

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

type SupportBundle struct {
	ID                           uuid.UUID           `db:"id"`
	CreatedAt                    time.Time           `db:"created_at"`
	OrganizationID               uuid.UUID           `db:"organization_id"`
	CustomerOrganizationID       uuid.UUID           `db:"customer_organization_id"`
	CreatedByUserAccountID       uuid.UUID           `db:"created_by_user_account_id"`
	Title                        string              `db:"title"`
	Description                  *string             `db:"description"`
	Status                       SupportBundleStatus `db:"status"`
	CollectTokenHash             []byte              `db:"collect_token_hash"`
	CollectTokenExpiresAt        *time.Time          `db:"collect_token_expires_at"`
	CollectCommand               *string             `db:"collect_command"`
	StatusChangedByUserAccountID *uuid.UUID          `db:"status_changed_by_user_account_id"`
	StatusChangedAt              *time.Time          `db:"status_changed_at"`
}

type SupportBundleWithDetails struct {
	SupportBundle
	CreatedByUserName        string     `db:"created_by_user_name"`
	CreatedByImageID         *uuid.UUID `db:"created_by_image_id"`
	CustomerOrganizationName string     `db:"customer_organization_name"`
	ResourceCount            int64      `db:"resource_count"`
	CommentCount             int64      `db:"comment_count"`
	LastCommentAt            *time.Time `db:"last_comment_at"`
	StatusChangedByUserName  *string    `db:"status_changed_by_user_name"`
	StatusChangedByImageID   *uuid.UUID `db:"status_changed_by_image_id"`
}

type SupportBundleResource struct {
	ID              uuid.UUID `db:"id"`
	CreatedAt       time.Time `db:"created_at"`
	SupportBundleID uuid.UUID `db:"support_bundle_id"`
	Name            string    `db:"name"`
	Content         string    `db:"content"`
}

type SupportBundleComment struct {
	ID              uuid.UUID `db:"id"`
	CreatedAt       time.Time `db:"created_at"`
	SupportBundleID uuid.UUID `db:"support_bundle_id"`
	UserAccountID   uuid.UUID `db:"user_account_id"`
	Content         string    `db:"content"`
}

type SupportBundleCommentWithUser struct {
	SupportBundleComment
	UserName    string     `db:"user_name"`
	UserImageID *uuid.UUID `db:"user_image_id"`
}
