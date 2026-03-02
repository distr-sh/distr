package supportbundle

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"text/template"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

var (
	//go:embed templates/*
	embeddedFS embed.FS

	templates *template.Template
)

func init() {
	fsys, err := fs.Sub(embeddedFS, "templates")
	if err != nil {
		panic(err)
	}
	templates = template.Must(template.New("").ParseFS(fsys, "*.sh"))
}

// GenerateCollectScript renders the collect-script.sh template with the given parameters.
func GenerateCollectScript(
	baseURL string,
	bundleID uuid.UUID,
	patKey string,
	envVars []types.SupportBundleConfigurationEnvVar,
) (string, error) {
	apiBase := fmt.Sprintf("%s/api/v1/support-bundle-collect/%s", baseURL, bundleID.String())
	authHeader := fmt.Sprintf("Authorization: AccessToken %s", patKey)

	data := map[string]any{
		"BundleID":   bundleID.String(),
		"BaseURL":    apiBase,
		"AuthHeader": authHeader,
		"EnvVars":    envVars,
	}

	var buf bytes.Buffer
	if err := templates.Lookup("collect-script.sh").Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
