package supportbundle

import (
	"bytes"
	"fmt"

	"github.com/distr-sh/distr/internal/resources"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// GenerateCollectScript renders the collect-script.sh template with the given parameters.
func GenerateCollectScript(
	baseURL string,
	bundleID uuid.UUID,
	bundleSecret string,
	envVars []types.SupportBundleConfigurationEnvVar,
) (string, error) {
	apiBase := fmt.Sprintf("%s/api/v1/support-bundle-collect/%s", baseURL, bundleID.String())

	data := map[string]any{
		"BundleID": bundleID.String(),
		"BaseURL":  apiBase,
		"Token":    bundleSecret,
		"EnvVars":  envVars,
	}

	tmpl, err := resources.GetTemplate("support-bundle/collect-script.sh")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
