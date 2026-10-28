package kubecollect

import (
	"testing"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

func parseObject(g *WithT, manifest string) map[string]any {
	var obj map[string]any
	g.Expect(yaml.Unmarshal([]byte(manifest), &obj)).To(Succeed())
	return obj
}

func TestSanitize_metadata(t *testing.T) {
	g := NewWithT(t)
	obj := parseObject(g, `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  managedFields:
    - manager: kubectl
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: '{"spec":{}}'
spec:
  template:
    spec:
      containers:
        - name: web
          env:
            - name: API_TOKEN
              value: inline
`)
	sanitize(obj, DefaultOptions())
	g.Expect(obj).To(Equal(parseObject(g, `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      containers:
        - name: web
          env:
            - name: API_TOKEN
              value: inline
`)))
}

func TestSanitize_secret(t *testing.T) {
	// "cGFzc3dvcmQ=" is "password", "AAEC" is three binary bytes.
	const secret = `
apiVersion: v1
kind: Secret
metadata:
  name: credentials
  annotations:
    team: shop
    kubectl.kubernetes.io/last-applied-configuration: '{"data":{"password":"cGFzc3dvcmQ="}}'
type: Opaque
data:
  password: cGFzc3dvcmQ=
  key: AAEC
`
	t.Run("without data", func(t *testing.T) {
		g := NewWithT(t)
		obj := parseObject(g, secret)
		sanitize(obj, Options{IncludeSecretData: false})
		g.Expect(obj).To(Equal(parseObject(g, `
apiVersion: v1
kind: Secret
metadata:
  name: credentials
  annotations:
    team: shop
type: Opaque
data:
  password: "[REDACTED, 8 bytes]"
  key: "[REDACTED, 3 bytes]"
`)))
	})

	t.Run("with data", func(t *testing.T) {
		g := NewWithT(t)
		obj := parseObject(g, secret)
		sanitize(obj, Options{IncludeSecretData: true})
		g.Expect(obj).NotTo(HaveKey("data"))
		g.Expect(obj).To(HaveKeyWithValue("stringData", map[string]any{
			"password": "password",
			"key":      "[BINARY, 3 bytes]",
		}))
	})

	for _, secretType := range []string{"helm.sh/release.v1", "kubernetes.io/service-account-token"} {
		t.Run(secretType+" never with data", func(t *testing.T) {
			g := NewWithT(t)
			obj := parseObject(g, `
apiVersion: v1
kind: Secret
metadata:
  name: release
type: `+secretType+`
data:
  release: cGFzc3dvcmQ=
`)
			sanitize(obj, Options{IncludeSecretData: true})
			g.Expect(obj).NotTo(HaveKey("stringData"))
			g.Expect(obj).To(HaveKeyWithValue("data", map[string]any{"release": "[REDACTED, 8 bytes]"}))
		})
	}
}

func TestSanitize_configMap(t *testing.T) {
	const configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
data:
  LOG_LEVEL: debug
binaryData:
  logo.png: AAEC
`
	t.Run("with data", func(t *testing.T) {
		g := NewWithT(t)
		obj := parseObject(g, configMap)
		sanitize(obj, Options{IncludeConfigMapData: true})
		g.Expect(obj).To(HaveKeyWithValue("data", map[string]any{"LOG_LEVEL": "debug"}))
		g.Expect(obj).To(HaveKeyWithValue("binaryData", map[string]any{"logo.png": "[REDACTED, 3 bytes]"}))
	})

	t.Run("without data", func(t *testing.T) {
		g := NewWithT(t)
		obj := parseObject(g, configMap)
		sanitize(obj, Options{IncludeConfigMapData: false})
		g.Expect(obj).To(HaveKeyWithValue("data", map[string]any{"LOG_LEVEL": "[REDACTED, 5 bytes]"}))
		g.Expect(obj).To(HaveKeyWithValue("binaryData", map[string]any{"logo.png": "[REDACTED, 3 bytes]"}))
	})
}
