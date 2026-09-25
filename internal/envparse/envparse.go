package envparse

import (
	"errors"
	"fmt"
	"net/mail"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

func PositiveDuration(value string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value)
	if err == nil && parsed.Nanoseconds() <= 0 {
		err = errors.New("duration must be positive")
	}
	return parsed, err
}

func ByteSlice(s string) ([]byte, error) {
	return []byte(s), nil
}

// Host accepts a host with an optional port. A scheme is rejected because the URL scheme of such a
// variable is taken from DISTR_HOST, and one given here would silently win over it.
func Host(value string) (string, error) {
	if value == "" || strings.Contains(value, "/") {
		return "", errors.New("must be a host with an optional port, without scheme or path")
	}
	return value, nil
}

func MailAddress(s string) (mail.Address, error) {
	if parsed, err := mail.ParseAddress(s); err != nil || parsed == nil {
		return mail.Address{}, err
	} else {
		return *parsed, nil
	}
}

func NonNegativeNumber(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err == nil && parsed < 0 {
		err = errors.New("number must not be negative")
	}
	return parsed, err
}

func PositiveNumber(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err == nil && parsed <= 0 {
		err = errors.New("number must be positive")
	}
	return parsed, err
}

func Float(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

func commaSeparated(value string) []string {
	var result []string
	for entry := range strings.SplitSeq(value, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			result = append(result, entry)
		}
	}
	return result
}

// EmailDomainList parses a comma-separated list of email domains, e.g. "example.com,spam.io".
func EmailDomainList(value string) ([]string, error) {
	domains := commaSeparated(strings.ToLower(value))
	for _, domain := range domains {
		if strings.ContainsAny(domain, "@/ ") {
			return nil, fmt.Errorf("invalid email domain %q", domain)
		}
	}
	return domains, nil
}

// IPPrefixList parses a comma-separated list of IP addresses and CIDR prefixes. A bare address
// is taken as the prefix covering only itself.
func IPPrefixList(value string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, entry := range commaSeparated(value) {
		if strings.Contains(entry, "/") {
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, err
			}
			prefixes = append(prefixes, prefix.Masked())
		} else if addr, err := netip.ParseAddr(entry); err != nil {
			return nil, err
		} else {
			addr = addr.Unmap().WithZone("")
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
		}
	}
	return prefixes, nil
}
