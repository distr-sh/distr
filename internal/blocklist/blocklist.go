package blocklist

import (
	"context"
	"net/netip"
	"slices"
	"strings"

	"github.com/distr-sh/distr/internal/env"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const (
	EmailBlockedMessage = "email addresses from this domain are not allowed"
	IPBlockedMessage    = "access from your network is not allowed"
)

// EmailBlocked reports whether the domain of the given address, or a domain it is a subdomain of,
// is listed in BLOCKED_EMAIL_DOMAINS.
func EmailBlocked(email string) bool {
	return emailBlocked(env.BlockedEmailDomains(), email)
}

func ClientIPBlocked(ctx context.Context) bool {
	return ipBlocked(env.BlockedIPs(), chimiddleware.GetClientIPAddr(ctx))
}

func emailBlocked(domains []string, email string) bool {
	return slices.ContainsFunc(strings.Split(email, "@")[1:], func(domain string) bool {
		return domainBlocked(domains, domain)
	})
}

func domainBlocked(domains []string, domain string) bool {
	domain = strings.TrimRight(strings.ToLower(strings.TrimSpace(domain)), ".")
	return slices.ContainsFunc(domains, func(blocked string) bool {
		return domain == blocked || strings.HasSuffix(domain, "."+blocked)
	})
}

func ipBlocked(prefixes []netip.Prefix, addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	addr = addr.Unmap().WithZone("")
	return slices.ContainsFunc(prefixes, func(prefix netip.Prefix) bool {
		return prefix.Contains(addr)
	})
}
