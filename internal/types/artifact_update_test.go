package types

import (
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestArtifactUpdatesAvailable(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	artifact := Artifact{ID: uuid.New(), VersioningStrategy: VersioningStrategySemver}
	publicArtifact := Artifact{ID: uuid.New(), VersioningStrategy: VersioningStrategySemver, Public: true}
	tag := func(artifact Artifact, name, digest string, createdAt time.Time) ArtifactVersion {
		version := artifactTag(name, digest, createdAt)
		version.ArtifactID = artifact.ID
		return version
	}
	v110 := tag(artifact, "1.1.0", "sha256:b", base)
	tags := []ArtifactVersion{
		tag(artifact, "1.2.0", "sha256:c", base.Add(-time.Hour)),
		v110,
		tag(artifact, "1.0.1", "sha256:a", base.Add(time.Hour)),
		tag(publicArtifact, "2.0.0", "sha256:e", base),
	}
	pinned, entitled, other := uuid.New(), uuid.New(), uuid.New()
	pull := func(customer uuid.UUID, artifact Artifact, version ArtifactVersion) CustomerArtifactPull {
		return CustomerArtifactPull{CustomerOrganizationID: customer, Artifact: artifact, ArtifactVersion: version}
	}
	pulls := []CustomerArtifactPull{
		pull(pinned, artifact, v110),
		pull(entitled, artifact, v110),
		pull(other, publicArtifact, tag(publicArtifact, "1.0.0", "sha256:d", base)),
	}

	g.Expect(updateNames(AvailableArtifactUpdates(pulls, tags, NewArtifactTagVisibility(false, nil)))).
		To(Equal([]string{"1.2.0", "1.2.0", "2.0.0"}), "the highest version wins over the most recent one")

	visibility := NewArtifactTagVisibility(true, []ArtifactEntitlementGrant{
		{CustomerOrganizationID: pinned, ArtifactID: artifact.ID, ManifestBlobDigest: new(Digest("sha256:b"))},
		{CustomerOrganizationID: entitled, ArtifactID: artifact.ID},
	})
	g.Expect(updateNames(AvailableArtifactUpdates(pulls, tags, visibility))).To(Equal([]string{"", "1.2.0", "2.0.0"}),
		"a customer entitled to one version has nothing newer to pull, and a public artifact is not gated")
}

func updateNames(updates []CustomerArtifactUpdate) []string {
	names := make([]string, len(updates))
	for i, update := range updates {
		if update.AvailableUpdate != nil {
			names[i] = update.AvailableUpdate.Name
		}
	}
	return names
}
