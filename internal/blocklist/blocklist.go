package blocklist

import (
	"net/http"
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

// RequestBlocked reports whether the remote address of the request or any address in its X-Forwarded-For header
// falls within one of the prefixes in BLOCKED_IPS. A client can prepend forged entries to the header but cannot
// remove the one a proxy appends, so every entry has to be checked rather than the one taken as the client IP.
func RequestBlocked(r *http.Request) bool {
	return requestBlocked(env.BlockedIPs(), r)
}

func requestBlocked(prefixes []netip.Prefix, r *http.Request) bool {
	if len(prefixes) == 0 {
		return false
	}
	if addrPort, err := netip.ParseAddrPort(r.RemoteAddr); err == nil && ipBlocked(prefixes, addrPort.Addr()) {
		return true
	}
	for _, header := range r.Header.Values("X-Forwarded-For") {
		for entry := range strings.SplitSeq(header, ",") {
			if addr, err := netip.ParseAddr(strings.TrimSpace(entry)); err == nil && ipBlocked(prefixes, addr) {
				return true
			}
		}
	}
	return false
}

func emailBlocked(domains []string, email string) bool {
	return slices.ContainsFunc(strings.Split(email, "@")[1:], func(domain string) bool {
		return domainBlocked(domains, domain)
	})
}

func domainBlocked(domains []string, domain string) bool {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
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
