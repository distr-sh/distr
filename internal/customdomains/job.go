package customdomains

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// verificationConcurrency bounds the DNS lookups in flight. Each one is capped at dns.LookupTimeout,
// so this is what keeps a large fleet of domains within the job timeout.
const verificationConcurrency = 8

// RunCustomDomainVerification refreshes the verification state every outbound URL is resolved
// against, so that no request ever has to wait for a DNS lookup.
func RunCustomDomainVerification(ctx context.Context) error {
	if !env.CustomDomainsConfigured() {
		return nil
	}

	domains, err := db.GetCustomDomainsDueForVerification(ctx, env.CustomDomainVerificationRefreshAfter())
	if err != nil {
		return fmt.Errorf("could not get custom domains due for verification: %w", err)
	}

	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(verificationConcurrency)
	for _, domain := range domains {
		group.Go(func() error {
			if err := verifyDomain(ctx, domain); err != nil {
				return fmt.Errorf("could not verify custom domain %v: %w", domain.Domain, err)
			}
			return nil
		})
	}
	return group.Wait()
}

func verifyDomain(ctx context.Context, domain types.CustomDomain) error {
	updated, _, err := CheckAndStore(ctx, domain)
	if err != nil {
		return err
	}
	// Only the transition is worth reporting: the fallback to the previous host is silent by design,
	// so this is what turns "our mails suddenly link somewhere else" into something answerable.
	if domain.Verified() && !updated.Verified() {
		err := fmt.Errorf("custom domain %v is no longer verified: %v", domain.Domain, *updated.VerificationError)
		internalctx.GetLogger(ctx).Warn("custom domain is no longer verified",
			zap.String("domain", domain.Domain),
			zap.String("organizationId", domain.OrganizationID.String()),
			zap.Error(err))
		// A job has no request hub in its context, and taking the one from there would panic.
		sentry.CurrentHub().CaptureException(err)
	}
	return nil
}
