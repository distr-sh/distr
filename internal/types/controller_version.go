package types

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
)

// A released revision must never change. During a rolling deployment, Distr already points targets at the new
// controller version while a replica of the previous release still serves manifests. Only a new revision, which
// that replica does not know, keeps it from rendering its own template for the new controller.
const (
	CurrentManifestFileRevision = "v4"
	CurrentComposeFileRevision  = "v2"
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
