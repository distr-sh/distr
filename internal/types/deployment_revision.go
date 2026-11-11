package types

import (
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type DeploymentRevisionTrigger string

const (
	// DeploymentRevisionTriggerUser is a revision a user created directly, by deploying or by
	// changing the deployment's configuration.
	DeploymentRevisionTriggerUser             DeploymentRevisionTrigger = "user"
	DeploymentRevisionTriggerAutomaticUpdate  DeploymentRevisionTrigger = "automatic_update"
	DeploymentRevisionTriggerSecretChange     DeploymentRevisionTrigger = "secret_change"
	DeploymentRevisionTriggerLicenseKeyChange DeploymentRevisionTrigger = "license_key_change"
)

type DeploymentRevision struct {
	Base
	DeploymentID           uuid.UUID                 `db:"deployment_id" json:"deploymentId"`
	ApplicationVersionID   uuid.UUID                 `db:"application_version_id" json:"applicationVersionId"`
	ValuesYaml             dbcrypto.Bytes            `db:"-" json:"valuesYaml,omitempty"`
	EnvFileData            dbcrypto.Bytes            `db:"-" json:"-"`
	ValuesHash             []byte                    `db:"values_hash" json:"-"`
	ForceRestart           bool                      `db:"force_restart" json:"forceRestart"`
	IgnoreRevisionSkew     bool                      `db:"ignore_revision_skew" json:"ignoreRevisionSkew"`
	HelmOptions            *HelmOptions              `db:"helm_options" json:"helmOptions,omitempty"`
	CreatedByUserAccountID *uuid.UUID                `db:"created_by_user_account_id" json:"-"`
	Trigger                DeploymentRevisionTrigger `db:"trigger"`
	// ChartDigest pins the Helm chart of the revision to the manifest digest its tag pointed to when the revision
	// was created. It is only known for OCI charts in the organization's own Distr registry.
	ChartDigest *Digest `db:"chart_digest"`
}

type HelmOptions struct {
	Timeout           Duration `db:"helm_options_timeout" json:"timeout"`
	WaitStrategy      string   `db:"helm_options_wait_strategy" json:"waitStrategy"`
	RollbackOnFailure bool     `db:"helm_options_rollback_on_failure" json:"rollbackOnFailure"`
	CleanupOnFailure  bool     `db:"helm_options_cleanup_on_failure" json:"cleanupOnFailure"`
	ForceConflicts    bool     `db:"helm_options_force_conflicts" json:"forceConflicts"`
}

// CustomerDeploymentRevision is a deployment revision of a customer organization with what its activity
// feed shows.
type CustomerDeploymentRevision struct {
	ID                     uuid.UUID `db:"id"`
	CreatedAt              time.Time `db:"created_at"`
	CustomerOrganizationID uuid.UUID `db:"customer_organization_id"`
	DeploymentTargetID     uuid.UUID `db:"deployment_target_id"`
	DeploymentTargetName   string    `db:"deployment_target_name"`
	ApplicationName        string    `db:"application_name"`
	ApplicationVersionName string    `db:"application_version_name"`
	// CreatedByName and CreatedByImageID are only set for a creator who is a member of the customer
	// organization, since a customer must not learn who works for the vendor.
	CreatedByName    *string    `db:"created_by_name"`
	CreatedByImageID *uuid.UUID `db:"created_by_image_id"`
}

// DeploymentRevisionCreator is the user who created a deployment revision, with their membership in the
// organization of the deployment target. Deleted is set when that user is no longer a member.
//
// It is scanned from a composite, which pgx decodes by position: the field order has to match
// deploymentRevisionCreatorOutputExpr in internal/db.
type DeploymentRevisionCreator struct {
	ID                     uuid.UUID  `db:"id"`
	Name                   *string    `db:"name"`
	Email                  string     `db:"email"`
	ImageID                *uuid.UUID `db:"image_id"`
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id"`
	PartnerOrganizationID  *uuid.UUID `db:"partner_organization_id"`
	Deleted                bool       `db:"deleted"`
}

// DeploymentRevisionWithCreator is a deployment revision enriched with the
// configuration needed to display it (application version, release name, docker
// type, values) and information about the user who created it.
type DeploymentRevisionWithCreator struct {
	ID                     uuid.UUID                  `db:"id"`
	CreatedAt              time.Time                  `db:"created_at"`
	ApplicationVersionID   uuid.UUID                  `db:"application_version_id"`
	ApplicationVersionName string                     `db:"application_version_name"`
	ReleaseName            *string                    `db:"release_name"`
	DockerType             *DockerType                `db:"docker_type"`
	ValuesYaml             dbcrypto.Bytes             `db:"values_yaml"`
	EnvFileData            dbcrypto.Bytes             `db:"env_file_data"`
	ForceRestart           bool                       `db:"force_restart"`
	IgnoreRevisionSkew     bool                       `db:"ignore_revision_skew"`
	HelmOptions            *HelmOptions               `db:"helm_options"`
	Trigger                DeploymentRevisionTrigger  `db:"trigger"`
	CreatedBy              *DeploymentRevisionCreator `db:"created_by"`
	LatestStatus           *DeploymentRevisionStatus  `db:"latest_status"`
}
