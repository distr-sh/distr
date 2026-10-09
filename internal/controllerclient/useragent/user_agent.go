package useragent

import "strings"

const (
	DistrControllerUserAgent = "DistrControllerClient"
	// legacyDistrAgentUserAgent is what controllers released before the rename to controller send.
	legacyDistrAgentUserAgent = "DistrAgentClient"
)

// ReportedVersion returns the version a controller reports in its User-Agent header.
func ReportedVersion(userAgent string) (string, bool) {
	for _, prefix := range []string{DistrControllerUserAgent, legacyDistrAgentUserAgent} {
		if version, ok := strings.CutPrefix(userAgent, prefix+"/"); ok {
			return version, true
		}
	}
	return "", false
}
