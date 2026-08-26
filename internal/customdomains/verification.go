package customdomains

import (
	"context"
	"errors"
	"slices"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dns"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
)

// verified reports whether a domain may be used for outbound URLs. When this instance has no CNAME
// target configured there is nothing a domain could be verified against — self-hosters point DNS at
// their own ingress — so every domain stays usable.
func verified(domain types.CustomDomain) bool {
	if !env.CustomDomainsConfigured() {
		return true
	}
	return domain.Verified()
}

func filterVerified(domains []types.CustomDomain) []types.CustomDomain {
	return slices.DeleteFunc(slices.Clone(domains), func(domain types.CustomDomain) bool {
		return !verified(domain)
	})
}

// Check looks up whether domain currently points at its expected target. A nil verificationError
// means it does. The message it carries is shown to the user, so it explains a problem with the
// record itself; a lookup that did not complete is something only the log can do anything with and
// is reported as inconclusive instead.
func Check(ctx context.Context, domain types.CustomDomain) (verificationError *string, inconclusive bool) {
	target := env.CustomDomainTarget()
	if target == nil {
		return new("no CNAME target is configured on this instance"), false
	}
	err := dns.VerifyCNAME(ctx, domain.Domain, *target)
	if err == nil {
		return nil, false
	} else if cnameErr, ok := errors.AsType[*dns.CNAMEError](err); ok {
		return new(cnameErr.Error()), false
	}
	internalctx.GetLogger(ctx).Warn("custom domain CNAME lookup failed",
		zap.Error(err), zap.String("domain", domain.Domain))
	return nil, true
}

// CheckAndStore runs a check and persists its outcome. An inconclusive lookup only records that the
// check ran: it says nothing about the record, so a domain that was working keeps working through a
// resolver outage instead of dropping out of every link, mail and agent manifest at once.
func CheckAndStore(
	ctx context.Context,
	domain types.CustomDomain,
) (updated *types.CustomDomain, inconclusive bool, err error) {
	verificationError, inconclusive := Check(ctx, domain)
	if inconclusive {
		updated, err = db.SetCustomDomainVerificationAttempted(ctx, domain.ID)
	} else {
		updated, err = db.SetCustomDomainVerificationResult(ctx, domain.ID, verificationError)
	}
	return updated, inconclusive, err
}
