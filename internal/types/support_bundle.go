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
)

type SupportBundleConfigurationEnvVar struct {
	OrganizationID uuid.UUID `db:"organization_id"`
	Name           string    `db:"name"`
	Redacted       bool      `db:"redacted"`
}

type SupportBundle struct {
	ID                      uuid.UUID           `db:"id"`
	CreatedAt               time.Time           `db:"created_at"`
	OrganizationID          uuid.UUID           `db:"organization_id"`
	CustomerOrganizationID  uuid.UUID           `db:"customer_organization_id"`
	CreatedByUserAccountID  uuid.UUID           `db:"created_by_user_account_id"`
	Title                   string              `db:"title"`
	Description             *string             `db:"description"`
	Status                  SupportBundleStatus `db:"status"`
	CollectTokenHash        []byte              `db:"collect_token_hash"`
	CollectTokenExpiresAt   *time.Time          `db:"collect_token_expires_at"`
	ResolvedByUserAccountID *uuid.UUID          `db:"resolved_by_user_account_id"`
}

type SupportBundleWithDetails struct {
	SupportBundle
	CreatedByUserName        string     `db:"created_by_user_name"`
	CreatedByImageID         *uuid.UUID `db:"created_by_image_id"`
	CustomerOrganizationName string     `db:"customer_organization_name"`
	ResourceCount            int64      `db:"resource_count"`
	CommentCount             int64      `db:"comment_count"`
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
