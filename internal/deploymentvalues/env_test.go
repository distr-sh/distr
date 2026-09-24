package deploymentvalues

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestEnvFileReplaceSecretsResolvesReference(t *testing.T) {
	g := NewWithT(t)
	deployment := renderAndHashAccessor{
		envFileData: []byte(`DATABASE_URL="postgres://distr:{{ .Secrets.POSTGRES_PASSWORD }}@db:5432/distr"` + "\n"),
	}

	data, err := EnvFileReplaceSecrets(deployment, []types.SecretWithUpdatedBy{
		testSecret("POSTGRES_PASSWORD", "hunter2"),
	}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal(`DATABASE_URL="postgres://distr:hunter2@db:5432/distr"` + "\n"))

	_, err = EnvFileReplaceSecrets(deployment, nil, nil)
	g.Expect(err).To(MatchError(ErrInvalidTemplate))
}
