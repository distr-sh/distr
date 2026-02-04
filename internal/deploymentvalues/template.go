package deploymentvalues

import (
	"bytes"
	"text/template"

	"github.com/distr-sh/distr/internal/types"
)

type templateData struct {
	Secrets map[string]string
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

func getTemplateData(secrets []types.SecretWithUpdatedBy, options ...templateDataOption) templateData {
	config := templateDataConfig{valueInterceptor: ident[string]}
	for _, opt := range options {
		opt(&config)
	}

	data := templateData{
		Secrets: make(map[string]string),
	}
	for _, secret := range secrets {
		data.Secrets[secret.Key] = config.valueInterceptor(secret.Value)
	}

	return data
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
