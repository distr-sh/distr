package supportbundle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/resources"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

const collectorReleasesURL = "https://github.com/distr-sh/distr/releases"

type CollectScriptParams struct {
	DeploymentType types.DeploymentType
	BaseURL        string
	BundleID       uuid.UUID
	BundleSecret   string
	// EnvVars are only collected by the Docker collect script. The Kubernetes one uploads the full
	// pod specs instead.
	EnvVars              []types.SupportBundleConfigurationEnvVar
	Scripts              []types.SupportBundleConfigurationScript
	LogTailLines         int
	ResourceMaxBytes     int
	ScriptTimeout        time.Duration
	CollectorDownloadURL string
}

// collectScriptCustomScript holds a custom script in the form the template needs. Every field is
// base64-encoded so that no vendor-controlled content can terminate the heredoc the collect script
// is written with, which would make the remainder of the rendered file run as the outer script.
type collectScriptCustomScript struct {
	NameBase64        string
	DescriptionBase64 string
	ContentBase64     string
}

// GenerateCollectScript renders the collect script of the deployment type with the given parameters.
func GenerateCollectScript(params CollectScriptParams) (string, error) {
	var templateName string
	switch params.DeploymentType {
	case types.DeploymentTypeDocker:
		templateName = "support-bundle/docker.sh"
	case types.DeploymentTypeKubernetes:
		templateName = "support-bundle/kubernetes.sh"
	default:
		return "", fmt.Errorf("%w: %v", types.ErrInvalidDeploymentType, params.DeploymentType)
	}

	apiBase := fmt.Sprintf("%s/api/v1/support-bundle-collect/%s", params.BaseURL, params.BundleID.String())

	scripts := make([]collectScriptCustomScript, len(params.Scripts))
	for i, script := range params.Scripts {
		var description string
		if script.Description != nil {
			description = *script.Description
		}
		scripts[i] = collectScriptCustomScript{
			NameBase64:        base64.StdEncoding.EncodeToString([]byte(script.Name)),
			DescriptionBase64: base64.StdEncoding.EncodeToString([]byte(description)),
			ContentBase64:     base64.StdEncoding.EncodeToString([]byte(script.Content)),
		}
	}

	// timeout(1) takes whole seconds, and rounding down could turn a configured limit into 0, which
	// disables the limit instead of tightening it.
	scriptTimeoutSeconds := int(math.Ceil(params.ScriptTimeout.Seconds()))
	if scriptTimeoutSeconds < 0 {
		scriptTimeoutSeconds = 0
	}

	data := map[string]any{
		"BundleID":             params.BundleID.String(),
		"BaseURL":              apiBase,
		"Token":                params.BundleSecret,
		"EnvVars":              params.EnvVars,
		"Scripts":              scripts,
		"LogTail":              params.LogTailLines,
		"ResourceMaxBytes":     params.ResourceMaxBytes,
		"ScriptTimeoutSeconds": scriptTimeoutSeconds,
		"CollectorDownloadURL": params.CollectorDownloadURL,
	}

	tmpl, err := resources.GetTemplate(templateName, "support-bundle/common.sh")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CollectorDownloadURL pins the Kubernetes collector to the release of this Distr, so that it speaks
// the same upload API. A development build has no release of its own and takes the latest one.
func CollectorDownloadURL() string {
	if buildconfig.IsRelease() {
		return collectorReleasesURL + "/download/" + buildconfig.Version()
	}
	return collectorReleasesURL + "/latest/download"
}
