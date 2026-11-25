package controllerhelm

var pullSecretKeys = []string{"imagePullSecrets", "pullSecrets"}

// AddImagePullSecret appends secretName to every imagePullSecrets and pullSecrets array in values, at any depth.
// An array that holds only strings gets the name as a string, any other array gets it as {name: secretName}.
func AddImagePullSecret(values map[string]any, secretName string) {
	for _, key := range pullSecretKeys {
		if s, ok := values[key].([]any); ok {
			values[key] = appendPullSecret(s, secretName)
		}
	}
	for _, v := range values {
		if m, ok := v.(map[string]any); ok {
			AddImagePullSecret(m, secretName)
		}
	}
}

func appendPullSecret(secrets []any, secretName string) []any {
	if len(secrets) == 0 || !allStrings(secrets) {
		// A map rather than corev1.LocalObjectReference, since chart templates read the entry as a map (.name).
		return append(secrets, map[string]any{"name": secretName})
	}
	// Vendors put an empty string into the array to choose the string form. The kubelet ignores empty names, but
	// the API server warns about them, so they are dropped.
	result := make([]any, 0, len(secrets)+1)
	for _, s := range secrets {
		if s != "" {
			result = append(result, s)
		}
	}
	return append(result, secretName)
}

func allStrings(values []any) bool {
	for _, v := range values {
		if _, ok := v.(string); !ok {
			return false
		}
	}
	return true
}
