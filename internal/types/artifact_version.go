package types

import (
	"time"

	"github.com/google/uuid"
)

type ArtifactVersion struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	CreatedAt              time.Time  `db:"created_at" json:"createdAt"`
	CreatedByUserAccountID *uuid.UUID `db:"created_by_useraccount_id" json:"-"`
	UpdatedAt              *time.Time `db:"updated_at" json:"updatedAt"`
	UpdatedByUserAccountID *uuid.UUID `db:"updated_by_useraccount_id" json:"-"`
	Name                   string     `db:"name" json:"name"`
	ManifestBlobDigest     Digest     `db:"manifest_blob_digest" json:"manifestBlobDigest"`
	ManifestBlobSize       int64      `db:"manifest_blob_size" json:"-"`
	ManifestContentType    string     `db:"manifest_content_type" json:"manifestContentType"`
	ManifestData           []byte     `db:"manifest_data" json:"-"`
	ArtifactID             uuid.UUID  `db:"artifact_id" json:"artifactId"`
}

// IsDigestVersion reports whether the version is named after its own manifest digest, which is the
// row a push creates alongside the tag so that the manifest can be pulled by digest.
func (av ArtifactVersion) IsDigestVersion() bool {
	return av.Name == string(av.ManifestBlobDigest)
}
