package customdomains

import (
	"context"
	"net/mail"
	"regexp"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var urlSchemeRegex = regexp.MustCompile("^https?://")

// AppDomainOrDefault resolves the effective app base URL for an organization:
// self-service CustomDomain first, then the legacy OrganizationBranding.app_domain
// (kept until the branding domain migration follow-up ticket), then env.Host().
func AppDomainOrDefault(ctx context.Context, orgID uuid.UUID, b *types.OrganizationBranding) string {
	domains := orgWideCustomDomains(ctx, orgID)
	if d := domainOfType(domains, types.DomainTypeApp); d != nil {
		return withScheme(*d)
	}
	if b != nil && b.AppDomain != nil {
		return withScheme(*b.AppDomain)
	}
	return env.Host()
}

// RegistryDomainOrDefault resolves the effective registry host for an organization:
// dedicated CustomDomain registry row first, then the CustomDomain app row (every
// custom domain serves registry traffic under /v2/ via the Caddy path routing), then
// the legacy OrganizationBranding.registry_domain, then env.RegistryHost().
func RegistryDomainOrDefault(ctx context.Context, orgID uuid.UUID, b *types.OrganizationBranding) string {
	domains := orgWideCustomDomains(ctx, orgID)
	if d := domainOfType(domains, types.DomainTypeRegistry); d != nil {
		return *d
	}
	if d := domainOfType(domains, types.DomainTypeApp); d != nil {
		return *d
	}
	if b != nil && b.RegistryDomain != nil {
		return *b.RegistryDomain
	}
	return env.RegistryHost()
}

func EmailFromAddressParsedOrDefault(b *types.OrganizationBranding) (*mail.Address, error) {
	if b != nil && b.EmailFromAddress != nil {
		return mail.ParseAddress(*b.EmailFromAddress)
	} else {
		return util.PtrTo(env.GetMailerConfig().FromAddress), nil
	}
}

// orgWideCustomDomains loads the organization's unscoped custom domains best-effort:
// on error the caller falls back to the legacy branding columns / instance defaults,
// which keep working for every organization.
func orgWideCustomDomains(ctx context.Context, orgID uuid.UUID) []types.CustomDomain {
	domains, err := db.GetOrgWideCustomDomains(ctx, orgID)
	if err != nil {
		internalctx.GetLogger(ctx).Warn("failed to resolve custom domains", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
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

// withScheme prefixes the domain with the scheme of env.Host() (https:// if the
// configured host has none) unless it already contains one. Self-service custom
// domains are stored as bare hostnames; legacy branding app domains may contain
// a scheme already.
func withScheme(domain string) string {
	if urlSchemeRegex.MatchString(domain) {
		return domain
	}
	scheme := urlSchemeRegex.FindString(env.Host())
	if scheme == "" {
		scheme = "https://"
	}
	return scheme + domain
}
