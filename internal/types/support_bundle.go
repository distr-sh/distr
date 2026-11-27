package types

import (
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type SupportBundleStatus string

const (
	SupportBundleStatusOpen     SupportBundleStatus = "open"
	SupportBundleStatusResolved SupportBundleStatus = "resolved"
)

type SupportBundleEvent string

const (
	SupportBundleEventCreated  SupportBundleEvent = "created"
	SupportBundleEventUploaded SupportBundleEvent = "uploaded"
	SupportBundleEventResolved SupportBundleEvent = "resolved"
	SupportBundleEventReopened SupportBundleEvent = "reopened"
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
	// StatusChangedAt is only set once a user resolved or reopened the bundle, so an open bundle that has it was
	// reopened.
	StatusChangedAt *time.Time `db:"status_changed_at"`
}

// CustomerSupportBundleEvent is the creation of a support bundle of a customer organization, the upload of its
// resources or the latest change of its status.
type CustomerSupportBundleEvent struct {
	SupportBundleID        uuid.UUID          `db:"support_bundle_id"`
	CustomerOrganizationID uuid.UUID          `db:"customer_organization_id"`
	CreatedAt              time.Time          `db:"created_at"`
	Title                  string             `db:"title"`
	Event                  SupportBundleEvent `db:"event"`
	User                   *ActivityUser      `db:"user_account"`
}

// SupportBundleWithDetails carries the customer and partner organization of the memberships the users who created
// the bundle and changed its status have in the vendor's organization. Deleted is set for a user who is no longer a
// member or whose account was deleted.
type SupportBundleWithDetails struct {
	SupportBundle
	CreatedByUserName                     *string    `db:"created_by_user_name"`
	CreatedByImageID                      *uuid.UUID `db:"created_by_image_id"`
	CreatedByCustomerOrganizationID       *uuid.UUID `db:"created_by_customer_organization_id"`
	CreatedByPartnerOrganizationID        *uuid.UUID `db:"created_by_partner_organization_id"`
	CreatedByDeleted                      bool       `db:"created_by_deleted"`
	CustomerOrganizationName              string     `db:"customer_organization_name"`
	ResourceCount                         int64      `db:"resource_count"`
	UploadedAt                            *time.Time `db:"uploaded_at"`
	CommentCount                          int64      `db:"comment_count"`
	LastCommentAt                         *time.Time `db:"last_comment_at"`
	StatusChangedByUserName               *string    `db:"status_changed_by_user_name"`
	StatusChangedByImageID                *uuid.UUID `db:"status_changed_by_image_id"`
	StatusChangedByCustomerOrganizationID *uuid.UUID `db:"status_changed_by_customer_organization_id"`
	StatusChangedByPartnerOrganizationID  *uuid.UUID `db:"status_changed_by_partner_organization_id"`
	StatusChangedByDeleted                bool       `db:"status_changed_by_deleted"`
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

// SupportBundleCommentWithUser carries the membership of the author like SupportBundleWithDetails.
type SupportBundleCommentWithUser struct {
	SupportBundleComment
	UserName                   *string    `db:"user_name"`
	UserImageID                *uuid.UUID `db:"user_image_id"`
	UserCustomerOrganizationID *uuid.UUID `db:"user_customer_organization_id"`
	UserPartnerOrganizationID  *uuid.UUID `db:"user_partner_organization_id"`
	UserDeleted                bool       `db:"user_deleted"`
}
