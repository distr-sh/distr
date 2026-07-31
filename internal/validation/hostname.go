package validation

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

// hostnamePattern matches RFC-1123 hostnames: dot-separated labels of at most 63 characters
// consisting of lowercase alphanumerics and hyphens (not at the start or end of a label),
// with at least two labels (a bare TLD is not a valid custom domain).
var hostnamePattern = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// NormalizeHostname brings a host value into the canonical form hostnames are validated, stored
// and looked up in: lower-case, without surrounding whitespace and without the trailing dot of a
// fully qualified domain name. A scheme, port or path is stripped, so that values which are not
// necessarily bare hostnames - request hosts, configured base URLs such as env.Host(), the legacy
// OrganizationBranding domain columns and user input pasted as a URL - all normalize to the same
// hostname.
func NormalizeHostname(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "://") {
		if u, err := url.Parse(value); err == nil && u.Host != "" {
			value = u.Hostname()
		}
	} else {
		value, _, _ = strings.Cut(value, "/")
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		}
	}
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
}

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
