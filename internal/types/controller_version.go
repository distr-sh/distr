package types

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
)

const (
	CurrentManifestFileRevision = "v3"
	CurrentComposeFileRevision  = "v2"
	// LegacyManifestFileRevision and LegacyComposeFileRevision are the newest revisions that name their resources
	// after the agent. A controller applies its own manifest by resource name without pruning, so a target set up
	// with one of them must keep receiving it (see DeploymentTarget.LegacyAgentManifest).
	LegacyManifestFileRevision = "v2"
	LegacyComposeFileRevision  = "v1"
)

var minVersionMultiDeployment = semver.MustParse("1.6.0")

type ControllerVersion struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	CreatedAt            time.Time `db:"created_at" json:"createdAt"`
	Name                 string    `db:"name" json:"name"`
	ManifestFileRevision string    `db:"manifest_file_revision" json:"-"`
	ComposeFileRevision  string    `db:"compose_file_revision" json:"-"`
}

func (cv ControllerVersion) CheckMultiDeploymentSupported() error {
	if cv.Name == "snapshot" {
		return nil
	}
	sv, err := semver.NewVersion(cv.Name)
	if err != nil {
		return err
	}
	if sv.LessThan(minVersionMultiDeployment) {
		return fmt.Errorf(
			"multi deployments not supported by controller version %v (requires %v)",
			sv, minVersionMultiDeployment,
		)
	}
	return nil
}
