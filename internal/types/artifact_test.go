package types

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func taggedVersion(createdAt time.Time, tags ...string) TaggedArtifactVersion {
	version := TaggedArtifactVersion{CreatedAt: createdAt}
	for _, tag := range tags {
		version.Tags = append(version.Tags, ArtifactVersionTag{Name: tag})
	}
	return version
}

func TestLatestArtifactTag(t *testing.T) {
	g := NewWithT(t)
	now := time.Now()
	signatureTag := "sha256-" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" + ".sig"
	v2 := taggedVersion(now, "2.0.0")
	v10 := taggedVersion(now.Add(-time.Hour), "1.10.0", "1.9.0")
	cosign := taggedVersion(now.Add(time.Hour), signatureTag)
	signature := taggedVersion(now.Add(2*time.Hour), "3.0.0")
	signature.InferredType = ManifestTypeSignature

	versions := []TaggedArtifactVersion{v10, v2, cosign, signature}
	latest, tag := LatestArtifactTag(versions)
	g.Expect(tag).To(Equal("2.0.0"), "signatures and their tags do not break SemVer ordering")
	g.Expect(ArtifactPullIsLatest(versions, latest, "2.0.0")).To(BeTrue())
	g.Expect(ArtifactPullIsLatest(versions, latest, "1.9.0")).To(BeFalse())
	g.Expect(ArtifactPullIsLatest(versions, latest, "0.1.0")).To(BeTrue(), "a tag no longer available counts as latest")

	_, tag = LatestArtifactTag([]TaggedArtifactVersion{v10, taggedVersion(now.Add(-2*time.Hour), "latest", "3.0.0")})
	g.Expect(tag).To(Equal("1.10.0"), "a tag that is not SemVer makes the creation date decide")
}
