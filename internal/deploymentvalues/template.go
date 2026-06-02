package deploymentvalues

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/types"
)

type templateData struct {
	Secrets     map[string]string
	LicenseKeys map[string]string
}

type templateDataConfig struct {
	valueInterceptor func(string) string
}

type templateDataOption = func(*templateDataConfig)

func chain[T any](fns ...func(T) T) func(T) T {
	return func(v T) T {
		for _, f := range fns {
			if f != nil {
				v = f(v)
			}
		}
		return v
	}
}

func ident[T any](v T) T { return v }

func withValueInterceptor(f func(string) string) templateDataOption {
	return func(config *templateDataConfig) {
		config.valueInterceptor = chain(config.valueInterceptor, f)
	}
}

func getTemplateData(
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
	options ...templateDataOption,
) (templateData, error) {
	config := templateDataConfig{valueInterceptor: ident[string]}
	for _, opt := range options {
		opt(&config)
	}

	data := templateData{
		Secrets:     make(map[string]string),
		LicenseKeys: make(map[string]string),
	}
	for _, secret := range secrets {
		data.Secrets[secret.Key] = config.valueInterceptor(secret.Value)
	}
	for _, lk := range licenseKeys {
		token, err := licensekey.GenerateToken(licensekey.FromLicenseKey(lk), env.Host())
		if err != nil {
			return templateData{}, fmt.Errorf("could not generate license key token for %q: %w", lk.Name, err)
		}
		data.LicenseKeys[lk.Name] = token
	}

	return data, nil
}

func parseTemplateBytes(name string, data []byte) (*template.Template, error) {
	return template.New(name).Option("missingkey=error").Parse(string(data))
}

func executeTemplate(tpl *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	} else {
		return buf.Bytes(), nil
	}
}
