package types

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentRevision struct {
	Base
	DeploymentID           uuid.UUID    `db:"deployment_id" json:"deploymentId"`
	ApplicationVersionID   uuid.UUID    `db:"application_version_id" json:"applicationVersionId"`
	ValuesYaml             []byte       `db:"-" json:"valuesYaml,omitempty"`
	EnvFileData            []byte       `db:"-" json:"-"`
	ValuesHash             []byte       `db:"values_hash" json:"-"`
	ForceRestart           bool         `db:"force_restart" json:"forceRestart"`
	IgnoreRevisionSkew     bool         `db:"ignore_revision_skew" json:"ignoreRevisionSkew"`
	HelmOptions            *HelmOptions `db:"helm_options" json:"helmOptions,omitempty"`
	CreatedByUserAccountID *uuid.UUID   `db:"created_by_user_account_id" json:"-"`
}

type HelmOptions struct {
	Timeout           Duration `db:"helm_options_timeout" json:"timeout"`
	WaitStrategy      string   `db:"helm_options_wait_strategy" json:"waitStrategy"`
	RollbackOnFailure bool     `db:"helm_options_rollback_on_failure" json:"rollbackOnFailure"`
	CleanupOnFailure  bool     `db:"helm_options_cleanup_on_failure" json:"cleanupOnFailure"`
	ForceConflicts    bool     `db:"helm_options_force_conflicts" json:"forceConflicts"`
}

// DeploymentRevisionWithCreator is a deployment revision enriched with the
// configuration needed to display it (application version, release name, docker
// type, values) and information about the user who created it.
type DeploymentRevisionWithCreator struct {
	ID                              uuid.UUID    `db:"id"`
	CreatedAt                       time.Time    `db:"created_at"`
	ApplicationVersionID            uuid.UUID    `db:"application_version_id"`
	ApplicationVersionName          string       `db:"application_version_name"`
	ReleaseName                     *string      `db:"release_name"`
	DockerType                      *DockerType  `db:"docker_type"`
	ValuesYaml                      []byte       `db:"values_yaml"`
	EnvFileData                     []byte       `db:"env_file_data"`
	ForceRestart                    bool         `db:"force_restart"`
	IgnoreRevisionSkew              bool         `db:"ignore_revision_skew"`
	HelmOptions                     *HelmOptions `db:"helm_options"`
	CreatedByID                     *uuid.UUID   `db:"created_by_id"`
	CreatedByName                   *string      `db:"created_by_name"`
	CreatedByEmail                  *string      `db:"created_by_email"`
	CreatedByImageID                *uuid.UUID   `db:"created_by_image_id"`
	CreatedByCustomerOrganizationID *uuid.UUID   `db:"created_by_customer_organization_id"`
	CreatedByPartnerOrganizationID  *uuid.UUID   `db:"created_by_partner_organization_id"`
	CreatedByDeleted                bool         `db:"created_by_deleted"`
}
