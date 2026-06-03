package deploymentvalues

import (
	"fmt"

	"github.com/distr-sh/distr/internal/types"
	"gopkg.in/yaml.v3"
)

type ValuesYAMLAccessor interface {
	GetValuesYAML() []byte
}

func ParsedValuesFileReplaceSecrets(
	d ValuesYAMLAccessor,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
) (map[string]any, error) {
	if data := d.GetValuesYAML(); data == nil {
		return nil, nil
	} else if tpl, err := parseTemplateBytes("valuesYaml", data); err != nil {
		return nil, fmt.Errorf("%w: deployment values file template parsing error: %w", ErrInvalidTemplate, err)
	} else if td, err := getTemplateData(secrets, licenseKeys); err != nil {
		return nil, fmt.Errorf("deployment values file template data error: %w", err)
	} else if data, err := executeTemplate(tpl, td); err != nil {
		return nil, fmt.Errorf("%w: deployment values file template execution error: %w", ErrInvalidTemplate, err)
	} else if result, err := parseDeploymentValuesYAML(data); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidTemplate, err)
	} else {
		return result, nil
	}
}

func ParsedValuesFile(d ValuesYAMLAccessor) (result map[string]any, err error) {
	if data := d.GetValuesYAML(); data == nil {
		return nil, nil
	} else {
		return parseDeploymentValuesYAML(data)
	}
}

func parseDeploymentValuesYAML(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("cannot parse Deployment values file: %w", err)
	}
	return result, nil
}
