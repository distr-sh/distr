package customdomains

import (
	"context"
	"fmt"
	"regexp"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var urlSchemeRegex = regexp.MustCompile("^https?://")

func AppDomainOrDefault(ctx context.Context, orgID uuid.UUID, b *types.OrganizationBranding) string {
	return appDomainOrDefault(customDomains(ctx, orgID, nil), b)
}

func appDomainOrDefault(vendorDomains []types.CustomDomain, b *types.OrganizationBranding) string {
	if d := domainOfType(vendorDomains, types.DomainTypeApp); d != nil {
		return withScheme(*d)
	}
	if b != nil && b.AppDomain != nil {
		return withScheme(*b.AppDomain)
	}
	return env.Host()
}

func AppDomain(ctx context.Context, orgID uuid.UUID) *string {
	return domainOfType(customDomains(ctx, orgID, nil), types.DomainTypeApp)
}

func CustomerPortalDomainOrDefault(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID *uuid.UUID,
	b *types.OrganizationBranding,
) string {
	return customerPortalDomainOrDefault(
		customerDomains(ctx, orgID, customerOrgID), customDomains(ctx, orgID, nil), b)
}

func customerPortalDomainOrDefault(
	customerScopedDomains, vendorDomains []types.CustomDomain,
	b *types.OrganizationBranding,
) string {
	if d := portalDomain(customerScopedDomains, vendorDomains); d != nil {
		return withScheme(*d)
	}
	// Falls through to the vendor's app domain (including its legacy branding fallback) rather than
	// straight to env.Host(), so a vendor without a portal domain still keeps its own hostname.
	return appDomainOrDefault(vendorDomains, b)
}

func CustomerPortalDomain(ctx context.Context, orgID uuid.UUID, customerOrgID *uuid.UUID) *string {
	return portalDomain(customerDomains(ctx, orgID, customerOrgID), customDomains(ctx, orgID, nil))
}

func portalDomain(customerScopedDomains, vendorDomains []types.CustomDomain) *string {
	if d := domainOfType(customerScopedDomains, types.DomainTypeCustomerPortal); d != nil {
		return d
	}
	return domainOfType(vendorDomains, types.DomainTypeCustomerPortal)
}

func customerDomains(ctx context.Context, orgID uuid.UUID, customerOrgID *uuid.UUID) []types.CustomDomain {
	if customerOrgID == nil {
		return nil
	}
	return customDomains(ctx, orgID, customerOrgID)
}

func RegistryDomainOrDefault(ctx context.Context, orgID uuid.UUID, b *types.OrganizationBranding) string {
	return registryDomainOrDefault(customDomains(ctx, orgID, nil), b)
}

func registryDomainOrDefault(vendorDomains []types.CustomDomain, b *types.OrganizationBranding) string {
	if d := domainOfType(vendorDomains, types.DomainTypeRegistry); d != nil {
		return *d
	}
	// Every custom domain serves registry traffic under /v2/ via the Caddy path routing, so the
	// app domain is a valid registry host too.
	if d := domainOfType(vendorDomains, types.DomainTypeApp); d != nil {
		return *d
	}
	if b != nil && b.RegistryDomain != nil {
		return *b.RegistryDomain
	}
	return env.RegistryHost()
}

// Errors are swallowed so callers fall back to the legacy branding columns / instance defaults
// instead of failing outright.
func customDomains(ctx context.Context, orgID uuid.UUID, customerOrgID *uuid.UUID) []types.CustomDomain {
	domains, err := db.GetCustomDomains(ctx, orgID, customerOrgID)
	if err != nil {
		internalctx.GetLogger(ctx).Warn("failed to resolve custom domains", zap.Error(err))
		return nil
	}
	return domains
}

func domainOfType(domains []types.CustomDomain, domainType types.DomainType) *string {
	for _, d := range domains {
		if d.Type == domainType {
			return &d.Domain
		}
	}
	return nil
}

// Legacy branding app domains may already contain a scheme; self-service ones never do.
func withScheme(domain string) string {
	if urlSchemeRegex.MatchString(domain) {
		return domain
	}
	return fmt.Sprintf("%v://%v", env.HostScheme(), domain)
}
