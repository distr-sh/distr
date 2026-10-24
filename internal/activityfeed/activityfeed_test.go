package activityfeed

import (
	"strings"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestArtifactVersionsSkipsVersionsWithoutReleaseTag(t *testing.T) {
	g := NewWithT(t)
	since := time.Now().Add(-Window)
	version := func(createdAt time.Time, inferredType types.ManifestType, tags ...string) types.TaggedArtifactVersion {
		v := types.TaggedArtifactVersion{ID: uuid.New(), CreatedAt: createdAt, InferredType: inferredType}
		for _, tag := range tags {
			v.Tags = append(v.Tags, types.ArtifactVersionTag{Name: tag})
		}
		return v
	}
	signatureTag := "sha256-" + strings.Repeat("a", 64) + ".sig"

	entries := ArtifactVersions(types.ArtifactWithDownloads{}, []types.TaggedArtifactVersion{
		version(since.Add(time.Hour), types.ManifestTypeContainerImage, "1.0.0", "latest"),
		version(since.Add(time.Hour), types.ManifestTypeContainerImage),
		version(since.Add(time.Hour), types.ManifestTypeSignature, signatureTag),
		version(since.Add(-time.Hour), types.ManifestTypeContainerImage, "0.9.0"),
	}, since)
	g.Expect(entries).To(ConsistOf(HaveField("Detail", HaveValue(Equal("1.0.0")))))
}
