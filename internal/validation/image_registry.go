package validation

import (
	"strings"

	"oras.land/oras-go/v2/registry"
)

// NormalizeImageRegistry strips surrounding whitespace, a scheme and trailing slashes, so that an image
// prefix pasted as a URL is accepted as well.
func NormalizeImageRegistry(value string) string {
	value = strings.TrimSpace(value)
	if _, rest, ok := strings.Cut(value, "://"); ok {
		value = rest
	}
	return strings.TrimRight(value, "/")
}

// ValidateImageRegistry validates an image prefix such as "ghcr.io/distr-sh/distr" that a fixed image name
// is appended to. The first component has to be a registry host, because a container runtime resolves a
// reference without one against Docker Hub.
func ValidateImageRegistry(value string) error {
	host, _, _ := strings.Cut(value, "/")
	if host != "localhost" && !strings.ContainsAny(host, ".:") {
		return NewValidationFailedError("must start with a registry host, e.g. registry.example.com/distr")
	}
	if ref, err := registry.ParseReference(value + "/docker-controller"); err != nil || ref.Reference != "" {
		return NewValidationFailedError(
			"must be a registry host followed by an optional lowercase path, without a tag or digest",
		)
	}
	return nil
}
