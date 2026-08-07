package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/distr-sh/distr/internal/validation"
)

// LookupTimeout bounds how long a CNAME verification may take, so a slow or unresponsive resolver
// cannot hang the caller. Mirrors internal/oidc/discovery.go's discoveryTimeout.
const LookupTimeout = 5 * time.Second

// VerifyCNAME reports whether domain currently resolves, via CNAME, to expectedTarget.
func VerifyCNAME(ctx context.Context, domain, expectedTarget string) (verified bool, detail string) {
	lookupCtx, cancel := context.WithTimeout(ctx, LookupTimeout)
	defer cancel()
	resolved, err := net.DefaultResolver.LookupCNAME(lookupCtx, domain)
	if err != nil {
		return false, fmt.Sprintf("could not resolve %s: %s", domain, err)
	}
	return compareCNAME(resolved, expectedTarget)
}

// compareCNAME compares a resolved CNAME against the expected target. Kept separate from the network
// lookup so it is unit testable without mocking DNS. LookupCNAME returns the queried name itself (not
// an error) when there is no CNAME record, e.g. a domain pointed at the target with a plain A record
// instead of a CNAME - normalizing and comparing here reports that case as a mismatch, not a false
// positive.
func compareCNAME(resolved, expected string) (verified bool, detail string) {
	resolvedHost := validation.NormalizeHostname(resolved)
	expectedHost := validation.NormalizeHostname(expected)
	if resolvedHost == expectedHost {
		return true, fmt.Sprintf("CNAME correctly points to %s", expected)
	}
	return false, fmt.Sprintf("CNAME points to %s instead of %s", resolvedHost, expected)
}
