package types

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CustomerOrganizationFeature string

const (
	CustomerOrganizationFeatureDeploymentTargets CustomerOrganizationFeature = "deployment_targets"
	CustomerOrganizationFeatureArtifacts         CustomerOrganizationFeature = "artifacts"
	CustomerOrganizationFeatureNotifications     CustomerOrganizationFeature = "notifications"
)

func ParseCustomerOrganizationFeature(value string) (CustomerOrganizationFeature, error) {
	switch value {
	case string(CustomerOrganizationFeatureDeploymentTargets):
		return CustomerOrganizationFeatureDeploymentTargets, nil
	case string(CustomerOrganizationFeatureArtifacts):
		return CustomerOrganizationFeatureArtifacts, nil
	default:
		return "", errors.New("invalid customer organization feature")
	}
}

type CustomerOrganization struct {
	ID             uuid.UUID                     `db:"id" json:"id"`
	CreatedAt      time.Time                     `db:"created_at" json:"createdAt"`
	OrganizationID uuid.UUID                     `db:"organization_id" json:"organizationId"`
	ImageID        *uuid.UUID                    `db:"image_id" json:"imageId,omitempty"`
	Name           string                        `db:"name" json:"name"`
	Features       []CustomerOrganizationFeature `db:"features" json:"features"`
}

type CustomerOrganizationWithUsage struct {
	CustomerOrganization
	UserCount             int64 `db:"user_count" json:"userCount"`
	DeploymentTargetCount int64 `db:"deployment_target_count" json:"deploymentTargetCount"`
}
