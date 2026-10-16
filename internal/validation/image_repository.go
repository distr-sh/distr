package validation

import (
	"strings"

	"oras.land/oras-go/v2/registry"
)

func NormalizeImageRepository(value string) string {
	value = strings.TrimSpace(value)
	if _, rest, ok := strings.Cut(value, "://"); ok {
		value = rest
	}
	return strings.TrimRight(value, "/")
}

func ValidateImageRepository(value string) error {
	host, _, _ := strings.Cut(value, "/")
	if host != "localhost" && !strings.ContainsAny(host, ".:") {
		return NewValidationFailedError(
			"the controller image repository must start with a registry host, e.g. registry.example.com/distr. " +
				"A host without a dot needs a port, e.g. registry:5000/distr",
		)
	}
	if ref, err := registry.ParseReference(value + "/docker-controller"); err != nil || ref.Reference != "" {
		return NewValidationFailedError("the controller image repository must be a registry host followed by " +
			"an optional lowercase path, without a tag or digest")
	}
	return nil
}
