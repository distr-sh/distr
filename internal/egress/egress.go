package egress

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
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

func CheckPublicHost(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if isNonPublic(ip) {
			return fmt.Errorf("%v is a non-public address", ip)
		}
		return nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("could not resolve %v: %w", host, err)
	}
	for _, addr := range addrs {
		if isNonPublic(addr.IP) {
			return fmt.Errorf("%v resolves to the non-public address %v", host, addr.IP)
		}
	}
	return nil
}

func rejectNonPublicAddress(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("could not parse address %v: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("could not parse address %v", address)
	}
	if isNonPublic(ip) {
		return fmt.Errorf("refusing to connect to the non-public address %v", ip)
	}
	return nil
}

func isNonPublic(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}
