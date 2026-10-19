package types

import (
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type ApplicationEntitlementBase struct {
	ID                     uuid.UUID        `db:"id" json:"id"`
	CreatedAt              time.Time        `db:"created_at" json:"createdAt"`
	Name                   string           `db:"name" json:"name"`
	ExpiresAt              *time.Time       `db:"expires_at" json:"expiresAt,omitempty"`
	ApplicationID          uuid.UUID        `db:"application_id" json:"applicationId"`
	OrganizationID         uuid.UUID        `db:"organization_id" json:"-"`
	CustomerOrganizationID *uuid.UUID       `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
	RegistryURL            *string          `db:"registry_url" json:"registryUrl,omitempty"`
	RegistryUsername       *dbcrypto.String `db:"registry_username" json:"registryUsername,omitempty"`
	RegistryPassword       *dbcrypto.String `db:"registry_password" json:"registryPassword,omitempty"`
	AllVersions            bool             `db:"all_versions" json:"allVersions"`
	AutoAddVersionRegex    *string          `db:"auto_add_version_regex" json:"autoAddVersionRegex,omitempty"`
}

type ApplicationEntitlementWithVersions struct {
	ApplicationEntitlementBase
	Versions []ApplicationVersion `db:"versions" json:"versions"`
}

type ApplicationEntitlement struct {
	ApplicationEntitlementWithVersions
	Application          Application           `db:"application" json:"application"`
	CustomerOrganization *CustomerOrganization `db:"customer_organization" json:"customerOrganization,omitempty"`
}

type ApplicationEntitlementAutoAddVersionRegex struct {
	ApplicationEntitlementID uuid.UUID `db:"id"`
	AutoAddVersionRegex      string    `db:"auto_add_version_regex"`
}

type DeploymentVersionUsage struct {
	DeploymentID           uuid.UUID `db:"deployment_id"`
	DeploymentTargetName   string    `db:"deployment_target_name"`
	ApplicationVersionID   uuid.UUID `db:"application_version_id"`
	ApplicationVersionName string    `db:"application_version_name"`
}

func (entitlement *ApplicationEntitlementWithVersions) CoversVersionWithID(id uuid.UUID) bool {
	return entitlement.AllVersions || slices.ContainsFunc(entitlement.Versions, func(v ApplicationVersion) bool {
		return v.ID == id
	})
}

// EntitledVersions narrows the versions of the application down to the ones the entitlement grants.
func (entitlement *ApplicationEntitlementWithVersions) EntitledVersions(
	applicationVersions []ApplicationVersion,
) []ApplicationVersion {
	if entitlement.AllVersions {
		return applicationVersions
	}
	return entitlement.Versions
}
