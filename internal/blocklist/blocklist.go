package blocklist

import (
	"net/netip"
	"slices"
	"strings"

	"github.com/distr-sh/distr/internal/env"
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

// IPBlocked reports whether the given address falls within one of the prefixes in BLOCKED_IPS.
// An invalid address is never blocked, so a request whose client IP could not be determined
// (e.g. behind a misconfigured proxy) is not refused for it.
func IPBlocked(addr netip.Addr) bool {
	return ipBlocked(env.BlockedIPs(), addr)
}

func emailBlocked(domains []string, email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	domain := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(email[at+1:])), ".")
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
