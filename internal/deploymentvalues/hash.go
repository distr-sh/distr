package deploymentvalues

import (
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/distr-sh/distr/internal/types"
)

type RenderAndHashAccessor interface {
	GetValuesYAML() []byte
	GetEnvFileData() []byte
}

func RenderAndHash(
	d RenderAndHashAccessor,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
) ([32]byte, error) {
	var result [32]byte
	h := sha256.New()

	if err := writeRenderedTemplateHashPart(
		h,
		"valuesYaml",
		d.GetValuesYAML(),
		secrets,
		licenseKeys,
		nil,
	); err != nil {
		return result, err
	}
	if err := writeRenderedTemplateHashPart(
		h,
		"envFileData",
		d.GetEnvFileData(),
		secrets,
		licenseKeys,
		escapeNewlines,
	); err != nil {
		return result, err
	}

	copy(result[:], h.Sum(nil))
	return result, nil
}

func writeRenderedTemplateHashPart(
	h hash.Hash,
	name string,
	data []byte,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
	valueInterceptor func(string) string,
) error {
	_, _ = h.Write([]byte(name))
	_, _ = h.Write([]byte{0})
	if len(data) == 0 {
		_, _ = h.Write([]byte{0})
		return nil
	}

	tpl, err := parseTemplateBytes(name, data)
	if err != nil {
		return fmt.Errorf("%w: %s template parsing error: %w", ErrInvalidTemplate, name, err)
	}
	options := []templateDataOption{}
	if valueInterceptor != nil {
		options = append(options, withValueInterceptor(valueInterceptor))
	}
	td, err := getTemplateData(secrets, licenseKeys, options...)
	if err != nil {
		return fmt.Errorf("%s template data error: %w", name, err)
	}
	rendered, err := executeTemplate(tpl, td)
	if err != nil {
		return fmt.Errorf("%w: %s template execution error: %w", ErrInvalidTemplate, name, err)
	}
	_, _ = h.Write(rendered)
	_, _ = h.Write([]byte{0})
	return nil
}
