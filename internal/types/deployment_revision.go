package types

import (
	"github.com/google/uuid"
)

type DeploymentRevision struct {
	Base
	DeploymentID         uuid.UUID    `db:"deployment_id" json:"deploymentId"`
	ApplicationVersionID uuid.UUID    `db:"application_version_id" json:"applicationVersionId"`
	ValuesYaml           []byte       `db:"-" json:"valuesYaml,omitempty"`
	EnvFileData          []byte       `db:"-" json:"-"`
	ForceRestart         bool         `db:"force_restart" json:"forceRestart"`
	IgnoreRevisionSkew   bool         `db:"ignore_revision_skew" json:"ignoreRevisionSkew"`
	HelmOptions          *HelmOptions `db:"helm_options" json:"helmOptions,omitempty"`
}

type HelmOptions struct {
	Timeout           Duration `db:"helm_options_timeout" json:"timeout"`
	WaitStrategy      string   `db:"helm_options_wait_strategy" json:"waitStrategy"`
	RollbackOnFailure bool     `db:"helm_options_rollback_on_failure" json:"rollbackOnFailure"`
	CleanupOnFailure  bool     `db:"helm_options_cleanup_on_failure" json:"cleanupOnFailure"`
}
