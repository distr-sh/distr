package handlers

import (
	"testing"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

type envFile []byte

func (f envFile) GetEnvFileData() []byte { return f }

func TestSecretsRedactor_masksSecretAsRenderedIntoEnvFile(t *testing.T) {
	g := NewWithT(t)
	secrets := []types.SecretWithUpdatedBy{
		{Secret: types.Secret{Key: "SERVICE_ACCOUNT", Value: dbcrypto.String("{\n  \"key\": \"abc\"\n}\n")}},
	}

	rendered, err := deploymentvalues.EnvFileReplaceSecrets(
		envFile("SERVICE_ACCOUNT={{ .Secrets.SERVICE_ACCOUNT }}\n"), secrets, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(secretsRedactor(secrets).Redact(string(rendered))).To(Equal("SERVICE_ACCOUNT=********\n"))
}
