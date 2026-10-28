package kubecollect

import (
	"encoding/base64"
	"fmt"
	"unicode"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
)

const lastAppliedConfigurationAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

// sanitize removes from an object what must not leave the customer's machine, or is only noise.
// The last-applied annotation is a full copy of the object as it was applied, so it would carry
// Secret and ConfigMap values past the rules below.
func sanitize(obj map[string]any, opts Options) {
	if metadata, ok := obj["metadata"].(map[string]any); ok {
		delete(metadata, "managedFields")
		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			delete(annotations, lastAppliedConfigurationAnnotation)
			if len(annotations) == 0 {
				delete(metadata, "annotations")
			}
		}
	}

	if obj["apiVersion"] != "v1" {
		return
	}
	switch obj["kind"] {
	case "Secret":
		sanitizeSecret(obj, opts.IncludeSecretData)
	case "ConfigMap":
		if !opts.IncludeConfigMapData {
			replaceValues(obj, "data", func(value string) string { return redacted(len(value)) })
		}
		replaceValues(obj, "binaryData", func(value string) string { return redacted(decodedLen(value)) })
	}
}

func sanitizeSecret(obj map[string]any, includeData bool) {
	switch corev1.SecretType(fmt.Sprint(obj["type"])) {
	case corev1.SecretTypeServiceAccountToken, corev1.SecretTypeBootstrapToken, "helm.sh/release.v1":
		includeData = false
	}
	if !includeData {
		replaceValues(obj, "data", func(value string) string { return redacted(decodedLen(value)) })
		return
	}
	data, ok := obj["data"].(map[string]any)
	if !ok {
		return
	}
	stringData := make(map[string]any, len(data))
	for key, value := range data {
		encoded, _ := value.(string)
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err != nil {
			stringData[key] = redacted(len(encoded))
		} else if !isText(decoded) {
			stringData[key] = fmt.Sprintf("[BINARY, %d bytes]", len(decoded))
		} else {
			stringData[key] = string(decoded)
		}
	}
	delete(obj, "data")
	obj["stringData"] = stringData
}

func replaceValues(obj map[string]any, field string, replace func(string) string) {
	if values, ok := obj[field].(map[string]any); ok {
		for key, value := range values {
			text, _ := value.(string)
			values[key] = replace(text)
		}
	}
}

func redacted(size int) string {
	return fmt.Sprintf("[REDACTED, %d bytes]", size)
}

func decodedLen(encoded string) int {
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		return len(decoded)
	}
	return len(encoded)
}

func isText(value []byte) bool {
	if !utf8.Valid(value) {
		return false
	}
	for _, r := range string(value) {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
