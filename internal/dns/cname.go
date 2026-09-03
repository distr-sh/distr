package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/distr-sh/distr/internal/validation"
)

// LookupTimeout bounds how long a CNAME verification may take, so a slow or unresponsive resolver
// cannot hang the caller. Mirrors internal/oidc/discovery.go's discoveryTimeout.
const LookupTimeout = 5 * time.Second

// CNAMEError reports that a domain does not point at the expected target, either because it has no
// CNAME record at all (Resolved is then empty) or because the record points somewhere else. A lookup
// that does not complete is deliberately not a CNAMEError: it says nothing about the record, so only
// a CNAMEError carries a message that means something to whoever has to create the record.
type CNAMEError struct {
	Domain   string
	Resolved string
	Expected string
}

func (e *CNAMEError) Error() string {
	if e.Resolved == "" {
		return fmt.Sprintf("no CNAME record found for %v, it must point to %v", e.Domain, e.Expected)
	}
	return fmt.Sprintf("CNAME points to %v instead of %v", e.Resolved, e.Expected)
}

// VerifyCNAME checks that domain resolves, via CNAME, to expectedTarget. A *CNAMEError means the
// record was read and does not point at the target. Any other error means the lookup did not
// complete and therefore says nothing about the record.
func VerifyCNAME(ctx context.Context, domain, expectedTarget string) error {
	lookupCtx, cancel := context.WithTimeout(ctx, LookupTimeout)
	defer cancel()
	resolved, err := net.DefaultResolver.LookupCNAME(lookupCtx, domain)
	if err != nil {
		if dnsErr, ok := errors.AsType[*net.DNSError](err); ok && dnsErr.IsNotFound {
			return &CNAMEError{Domain: validation.NormalizeHostname(domain), Expected: expectedTarget}
		}
		return fmt.Errorf("could not resolve %v: %w", domain, err)
	}
	return compareCNAME(domain, resolved, expectedTarget)
}

// compareCNAME compares a resolved CNAME against the expected target. Kept separate from the network
// lookup so it is unit testable without mocking DNS. LookupCNAME returns the queried name itself (not
// an error) when the name exists but has no CNAME record, e.g. a domain pointed at the target with a
// plain A record - comparing against the queried name here reports that case as a missing record
// rather than as a CNAME pointing at the domain itself.
func compareCNAME(domain, resolved, expected string) error {
	domainHost := validation.NormalizeHostname(domain)
	resolvedHost := validation.NormalizeHostname(resolved)
	switch resolvedHost {
	case validation.NormalizeHostname(expected):
		return nil
	case domainHost:
		return &CNAMEError{Domain: domainHost, Expected: expected}
	default:
		return &CNAMEError{Domain: domainHost, Resolved: resolvedHost, Expected: expected}
	}
}
