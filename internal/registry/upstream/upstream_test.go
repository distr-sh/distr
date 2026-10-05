package upstream

import (
	"testing"

	. "github.com/onsi/gomega"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestParseManifest(t *testing.T) {
	g := NewWithT(t)
	data := []byte(`{
		"schemaVersion": 2,
		"mediaType": "application/vnd.oci.image.manifest.v1+json",
		"config": {
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			"size": 2
		},
		"layers": []
	}`)

	blobs, _, err := parseManifest(data, ocispec.MediaTypeImageManifest, godigest.FromBytes(data))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(blobs).To(HaveLen(1))

	_, _, err = parseManifest(data, ocispec.MediaTypeImageManifest, godigest.FromString("other"))
	g.Expect(err).To(MatchError(ContainSubstring("does not match its digest")))

	metadata := []byte(`{"Code":"Success","AccessKeyId":"AKIA","SecretAccessKey":"secret"}`)
	for _, contentType := range []string{"text/plain", "application/json"} {
		_, _, err = parseManifest(metadata, contentType, godigest.FromBytes(metadata))
		g.Expect(err).To(MatchError(ContainSubstring("unsupported manifest media type")))
	}
}
