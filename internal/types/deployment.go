package types

import (
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type Deployment struct {
	Base
	DeploymentTargetID                 uuid.UUID   `db:"deployment_target_id"`
	ReleaseName                        *string     `db:"release_name"`
	ApplicationEntitlementID           *uuid.UUID  `db:"application_entitlement_id"`
	DockerType                         *DockerType `db:"docker_type"`
	AutomaticApplicationUpdatesEnabled bool        `db:"automatic_application_updates_enabled"`
}

type DeploymentWithLatestRevision struct {
	Deployment
	DeploymentRevisionID        uuid.UUID `db:"deployment_revision_id"`
	DeploymentRevisionCreatedAt time.Time `db:"deployment_revision_created_at"`
	// Deprecated: Use [DeploymentWithLatestRevision.Application] instead
	ApplicationID uuid.UUID `db:"application_id"`
	// Deprecated: Use [DeploymentWithLatestRevision.Application] instead
	ApplicationName         string                    `db:"application_name"`
	Application             Application               `db:"application"`
	ApplicationVersionID    uuid.UUID                 `db:"application_version_id"`
	ApplicationVersionName  string                    `db:"application_version_name"`
	ApplicationLinkTemplate string                    `db:"application_link_template"`
	ApplicationLink         string                    `db:"-"`
	ValuesYaml              dbcrypto.Bytes            `db:"values_yaml"`
	EnvFileData             dbcrypto.Bytes            `db:"env_file_data"`
	ValuesHash              []byte                    `db:"values_hash"`
	LatestStatus            *DeploymentRevisionStatus `db:"latest_status"`
	ForceRestart            bool                      `db:"force_restart"`
	IgnoreRevisionSkew      bool                      `db:"ignore_revision_skew"`
	HelmOptions             *HelmOptions              `db:"helm_options"`
	ChartDigest             *Digest                   `db:"chart_digest"`
	// CurrentDeploymentRevisionID is the revision a controller last reported as applied, which differs
	// from DeploymentRevisionID while a newer revision is being rolled out or has failed.
	CurrentDeploymentRevisionID   *uuid.UUID                 `db:"current_deployment_revision_id"`
	CurrentStatus                 *DeploymentRevisionStatus  `db:"current_status"`
	CurrentApplicationVersionID   *uuid.UUID                 `db:"current_application_version_id"`
	CurrentApplicationVersionName *string                    `db:"current_application_version_name"`
	DeploymentRevisionCreatedBy   *DeploymentRevisionCreator `db:"deployment_revision_created_by"`
}

// NewestStatus returns the status the controller reported last, no matter which revision it belongs to.
func (d *DeploymentWithLatestRevision) NewestStatus() *DeploymentRevisionStatus {
	if d.LatestStatus == nil {
		return d.CurrentStatus
	}
	if d.CurrentStatus == nil || d.LatestStatus.CreatedAt.After(d.CurrentStatus.CreatedAt) {
		return d.LatestStatus
	}
	return d.CurrentStatus
}

func (d *DeploymentWithLatestRevision) GetValuesYAML() []byte {
	return d.ValuesYaml
}

func (d *DeploymentWithLatestRevision) GetEnvFileData() []byte {
	return d.EnvFileData
}
