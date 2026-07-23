package validation

import (
	"regexp"
)

// hostnamePattern matches RFC-1123 hostnames: dot-separated labels of at most 63 characters
// consisting of lowercase alphanumerics and hyphens (not at the start or end of a label),
// with at least two labels (a bare TLD is not a valid custom domain).
var hostnamePattern = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ValidateHostname validates a bare lowercase hostname (no scheme, no port, no trailing dot).
func ValidateHostname(hostname string) error {
	if len(hostname) > 253 {
		return NewValidationFailedError("hostname is too long")
	}
	if !hostnamePattern.MatchString(hostname) {
		return NewValidationFailedError("invalid hostname format")
	}
	return nil
}
