package controllerhelm

import (
	"testing"

	. "github.com/onsi/gomega"
)

const secretName = "sh.distr.agent.v1.release.pull"

var secretRef = map[string]any{"name": secretName}

func TestAddImagePullSecret(t *testing.T) {
	g := NewWithT(t)
	values := map[string]any{
		"imagePullSecrets": []any{map[string]any{"name": "foo"}},
		"server": map[string]any{
			"imagePullSecrets": []any{},
			"image":            map[string]any{"pullSecrets": []any{""}},
		},
		"worker": map[string]any{"imagePullSecrets": []any{"foo", ""}},
		"db":     map[string]any{"imagePullSecrets": map[string]any{}},
	}

	AddImagePullSecret(values, secretName)

	g.Expect(values).To(Equal(map[string]any{
		"imagePullSecrets": []any{map[string]any{"name": "foo"}, secretRef},
		"server": map[string]any{
			"imagePullSecrets": []any{secretRef},
			"image":            map[string]any{"pullSecrets": []any{secretName}},
		},
		"worker": map[string]any{"imagePullSecrets": []any{"foo", secretName}},
		"db":     map[string]any{"imagePullSecrets": map[string]any{}},
	}))
}

func TestAddImagePullSecretMixedArray(t *testing.T) {
	g := NewWithT(t)
	values := map[string]any{"imagePullSecrets": []any{"", map[string]any{"name": "foo"}}}

	AddImagePullSecret(values, secretName)

	g.Expect(values["imagePullSecrets"]).To(Equal([]any{"", map[string]any{"name": "foo"}, secretRef}))
}
