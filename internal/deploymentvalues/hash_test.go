package deploymentvalues

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

type renderAndHashAccessor struct {
	valuesYAML  []byte
	envFileData []byte
}

func (a renderAndHashAccessor) GetValuesYAML() []byte {
	return a.valuesYAML
}

func (a renderAndHashAccessor) GetEnvFileData() []byte {
	return a.envFileData
}

func TestRenderAndHashReferencedSecretChange(t *testing.T) {
	g := NewWithT(t)
	deployment := renderAndHashAccessor{
		valuesYAML:  []byte("apiToken: '{{ .Secrets.apiToken }}'\n"),
		envFileData: []byte("STATIC=value\n"),
	}

	oldHash, err := RenderAndHash(deployment, []types.SecretWithUpdatedBy{
		testSecret("apiToken", "old"),
		testSecret("unused", "same"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	newHash, err := RenderAndHash(deployment, []types.SecretWithUpdatedBy{
		testSecret("apiToken", "new"),
		testSecret("unused", "same"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(newHash).NotTo(Equal(oldHash))
}

func TestRenderAndHashUnrelatedSecretChange(t *testing.T) {
	g := NewWithT(t)
	deployment := renderAndHashAccessor{
		valuesYAML:  []byte("apiToken: '{{ .Secrets.apiToken }}'\n"),
		envFileData: []byte("STATIC=value\n"),
	}

	oldHash, err := RenderAndHash(deployment, []types.SecretWithUpdatedBy{
		testSecret("apiToken", "same"),
		testSecret("unused", "old"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	newHash, err := RenderAndHash(deployment, []types.SecretWithUpdatedBy{
		testSecret("apiToken", "same"),
		testSecret("unused", "new"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(newHash).To(Equal(oldHash))
}

func TestRenderAndHashEnvFileEscapesSecretNewlines(t *testing.T) {
	g := NewWithT(t)
	deployment := renderAndHashAccessor{
		envFileData: []byte("API_TOKEN={{ .Secrets.apiToken }}\n"),
	}

	hashWithEscapedNewline, err := RenderAndHash(deployment, []types.SecretWithUpdatedBy{
		testSecret("apiToken", "line1\nline2"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	hashWithLiteralEscape, err := RenderAndHash(deployment, []types.SecretWithUpdatedBy{
		testSecret("apiToken", "line1\\nline2"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(hashWithEscapedNewline).To(Equal(hashWithLiteralEscape))
}

func testSecret(key, value string) types.SecretWithUpdatedBy {
	return types.SecretWithUpdatedBy{Secret: types.Secret{Key: key, Value: value}}
}
