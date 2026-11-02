package types

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type CustomerOrganization struct {
	ID                    uuid.UUID                     `db:"id" json:"id"`
	CreatedAt             time.Time                     `db:"created_at" json:"createdAt"`
	OrganizationID        uuid.UUID                     `db:"organization_id" json:"organizationId"`
	ImageID               *uuid.UUID                    `db:"image_id" json:"imageId,omitempty"`
	Name                  string                        `db:"name" json:"name"`
	Features              []CustomerOrganizationFeature `db:"features" json:"features"`
	PartnerOrganizationID *uuid.UUID                    `db:"partner_organization_id" json:"partnerOrganizationId,omitempty"` //nolint:lll
}

func (co *CustomerOrganization) HasFeature(feature CustomerOrganizationFeature) bool {
	return slices.Contains(co.Features, feature)
}

type CustomerOrganizationWithUsage struct {
	CustomerOrganization
	UserCount             int64 `db:"user_count" json:"userCount"`
	DeploymentTargetCount int64 `db:"deployment_target_count" json:"deploymentTargetCount"`
}
