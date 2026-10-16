package egress

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"syscall"
	"time"
)

// Taken from the IANA IPv4 and IPv6 special-purpose address registries. IPv6 is restricted to the global
// unicast range instead of listing what lies outside it.
var (
	nonPublicIPv4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("169.254.0.0/16"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.31.196.0/24"),
		netip.MustParsePrefix("192.52.193.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("192.175.48.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
	}
	globalUnicastIPv6 = netip.MustParsePrefix("2000::/3")
	nonPublicIPv6     = []netip.Prefix{
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("2002::/16"),
		netip.MustParsePrefix("2620:4f:8000::/48"),
		netip.MustParsePrefix("3fff::/20"),
	}
)

func Transport(allowNonPublic bool) *http.Transport {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if !allowNonPublic {
		dialer.Control = rejectNonPublicAddress
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = dialer.DialContext
	return transport
}

// CheckPublicHost does not replace the check of Transport, since a DNS record may change between this
// lookup and the connection.
func CheckPublicHost(ctx context.Context, host string) error {
	if addr, err := netip.ParseAddr(host); err == nil {
		if !isPublic(addr) {
			return fmt.Errorf("%v is a non-public address", addr)
		}
		return nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("could not resolve %v: %w", host, err)
	}
	for _, addr := range addrs {
		if !isPublic(addr) {
			return fmt.Errorf("%v resolves to the non-public address %v", host, addr)
		}
	}
	return nil
}

func rejectNonPublicAddress(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("could not parse address %v: %w", address, err)
	}
	if !isPublic(addrPort.Addr()) {
		return fmt.Errorf("refusing to connect to the non-public address %v", addrPort.Addr())
	}
	return nil
}

func isPublic(addr netip.Addr) bool {
	addr = addr.Unmap()
	contains := func(p netip.Prefix) bool { return p.Contains(addr) }
	if addr.Is4() {
		return !slices.ContainsFunc(nonPublicIPv4, contains)
	}
	return globalUnicastIPv6.Contains(addr) && !slices.ContainsFunc(nonPublicIPv6, contains)
}
