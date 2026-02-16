package types

import (
	"time"

	"github.com/google/uuid"
)

type ApplicationLicenseBase struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	Name                   string     `db:"name" json:"name"`
	ExpiresAt              *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
	ApplicationID          uuid.UUID  `db:"application_id" json:"applicationId"`
	OrganizationID         uuid.UUID  `db:"organization_id" json:"-"`
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
	RegistryURL            *string    `db:"registry_url" json:"registryUrl,omitempty"`
	RegistryUsername       *string    `db:"registry_username" json:"registryUsername,omitempty"`
	RegistryPassword       *string    `db:"registry_password" json:"registryPassword,omitempty"`
}

type ApplicationLicenseWithVersions struct {
	ApplicationLicenseBase
	Versions []ApplicationVersion `db:"versions" json:"versions"`
}

type ApplicationLicense struct {
	ApplicationLicenseWithVersions
	Application          Application           `db:"application" json:"application"`
	CustomerOrganization *CustomerOrganization `db:"customer_organization" json:"customerOrganization,omitempty"`
}

type DeploymentVersionUsage struct {
	DeploymentID           uuid.UUID `db:"deployment_id"`
	DeploymentTargetName   string    `db:"deployment_target_name"`
	ApplicationVersionID   uuid.UUID `db:"application_version_id"`
	ApplicationVersionName string    `db:"application_version_name"`
}

func (license *ApplicationLicenseWithVersions) HasVersionWithID(id uuid.UUID) bool {
	for _, v := range license.Versions {
		if v.ID == id {
			return true
		}
	}
	return false
}
