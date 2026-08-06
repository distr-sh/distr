package validation

import (
	"regexp"
	"strings"
)

// SlugMaxLength is the maximum length of a slug. The organization slug is a path segment of an OCI
// repository name, which is where the format comes from: lowercase alphanumeric groups, separated by
// a period, one or two underscores, or one or more hyphens.
const SlugMaxLength = 64

var slugPattern = regexp.MustCompile(`^[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*$`)

func NormalizeSlug(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return NewValidationFailedError(
			"a slug must consist of lowercase letters and digits, separated by '.', '_', '__' or '-'")
	}
	if len(slug) > SlugMaxLength {
		return NewValidationFailedError("a slug must be at most 64 characters")
	}
	return nil
}
