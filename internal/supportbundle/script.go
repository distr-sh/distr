package supportbundle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/distr-sh/distr/internal/resources"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type CollectScriptParams struct {
	BaseURL              string
	BundleID             uuid.UUID
	BundleSecret         string
	EnvVars              []types.SupportBundleConfigurationEnvVar
	Scripts              []types.SupportBundleConfigurationScript
	LogTailLines         int
	ScriptOutputMaxBytes int
}

// collectScriptCustomScript holds a custom script in the form the template needs: the name and
// description as ready-to-use shell literals, and the body base64-encoded so that no content can
// terminate the heredoc the collect script is written with.
type collectScriptCustomScript struct {
	NameQuoted        string
	DescriptionQuoted string
	ContentBase64     string
}

// GenerateCollectScript renders the collect-script.sh template with the given parameters.
func GenerateCollectScript(params CollectScriptParams) (string, error) {
	apiBase := fmt.Sprintf("%s/api/v1/support-bundle-collect/%s", params.BaseURL, params.BundleID.String())

	scripts := make([]collectScriptCustomScript, len(params.Scripts))
	for i, script := range params.Scripts {
		var description string
		if script.Description != nil {
			description = *script.Description
		}
		scripts[i] = collectScriptCustomScript{
			NameQuoted:        shellQuote(script.Name),
			DescriptionQuoted: shellQuote(description),
			ContentBase64:     base64.StdEncoding.EncodeToString([]byte(script.Content)),
		}
	}

	data := map[string]any{
		"BundleID":             params.BundleID.String(),
		"BaseURL":              apiBase,
		"Token":                params.BundleSecret,
		"EnvVars":              params.EnvVars,
		"Scripts":              scripts,
		"LogTailLines":         params.LogTailLines,
		"ScriptOutputMaxBytes": params.ScriptOutputMaxBytes,
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

// shellQuote turns s into a single-quoted shell literal. A single quote cannot be escaped inside
// single quotes, so every occurrence ends the literal, appends an escaped quote and starts a new one.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
