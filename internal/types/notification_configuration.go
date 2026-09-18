package types

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// NotificationRecipient is a user account selected to receive a notification, together with the
// membership that decides what it may see of the notification's subject.
type NotificationRecipient struct {
	ID                     uuid.UUID  `db:"id"`
	Email                  string     `db:"email"`
	Name                   string     `db:"name"`
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id"`
	PartnerOrganizationID  *uuid.UUID `db:"partner_organization_id"`
}

type NotificationApplication struct {
	ID      uuid.UUID      `db:"id"`
	Name    string         `db:"name"`
	Type    DeploymentType `db:"type"`
	ImageID *uuid.UUID     `db:"image_id"`
}

type NotificationArtifact struct {
	ID      uuid.UUID  `db:"id"`
	Name    string     `db:"name"`
	ImageID *uuid.UUID `db:"image_id"`
}

// DeploymentPendingUpdate is a deployment that does not run the version a notification announces.
type DeploymentPendingUpdate struct {
	DeploymentID             uuid.UUID  `db:"deployment_id"`
	ReleaseName              *string    `db:"release_name"`
	DeploymentTargetID       uuid.UUID  `db:"deployment_target_id"`
	DeploymentTargetName     string     `db:"deployment_target_name"`
	CustomerOrganizationID   *uuid.UUID `db:"customer_organization_id"`
	CustomerOrganizationName *string    `db:"customer_organization_name"`
	PartnerOrganizationID    *uuid.UUID `db:"partner_organization_id"`
	CurrentVersionName       string     `db:"current_version_name"`
	// Entitled reports whether the entitlement the deployment was created with covers the
	// announced version. It is always true for a deployment without an entitlement.
	Entitled bool `db:"entitled"`
}

// ArtifactVersionEntitlement is who may know about an artifact version. An organization that has
// not configured any artifact entitlement at all gates nothing, in which case every customer is
// entitled to every version.
type ArtifactVersionEntitlement struct {
	Gated                   bool
	CustomerOrganizationIDs []uuid.UUID
}

func (e ArtifactVersionEntitlement) Allows(customerOrganizationID uuid.UUID) bool {
	if !e.Gated {
		return true
	}
	return slices.Contains(e.CustomerOrganizationIDs, customerOrganizationID)
}

type ApplicationNotificationConfiguration struct {
	ID                            uuid.UUID  `db:"id"`
	CreatedAt                     time.Time  `db:"created_at"`
	OrganizationID                uuid.UUID  `db:"organization_id"`
	CustomerOrganizationID        *uuid.UUID `db:"customer_organization_id"`
	Name                          string     `db:"name"`
	Enabled                       bool       `db:"enabled"`
	UpdateAvailableTriggerEnabled bool       `db:"update_available_trigger_enabled"`
	CustomerMessage               *string    `db:"customer_message"`

	Applications []NotificationApplication `db:"applications"`
	Recipients   []NotificationRecipient   `db:"recipients"`

	// ApplicationIDs and UserAccountIDs are what insert and update write to the link tables. The
	// selected rows come back in Applications and Recipients instead.
	ApplicationIDs []uuid.UUID `db:"-"`
	UserAccountIDs []uuid.UUID `db:"-"`
}

type ArtifactNotificationConfiguration struct {
	ID                       uuid.UUID  `db:"id"`
	CreatedAt                time.Time  `db:"created_at"`
	OrganizationID           uuid.UUID  `db:"organization_id"`
	CustomerOrganizationID   *uuid.UUID `db:"customer_organization_id"`
	Name                     string     `db:"name"`
	Enabled                  bool       `db:"enabled"`
	NewVersionTriggerEnabled bool       `db:"new_version_trigger_enabled"`
	CustomerMessage          *string    `db:"customer_message"`

	Artifacts  []NotificationArtifact  `db:"artifacts"`
	Recipients []NotificationRecipient `db:"recipients"`

	ArtifactIDs    []uuid.UUID `db:"-"`
	UserAccountIDs []uuid.UUID `db:"-"`
}
