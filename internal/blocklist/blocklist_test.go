package blocklist

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/distr-sh/distr/internal/envparse"
	. "github.com/onsi/gomega"
)

func TestEmailBlocked(t *testing.T) {
	g := NewWithT(t)
	domains, err := envparse.EmailDomainList(" Example.com, ,spam.io,fqdn.org.")
	g.Expect(err).NotTo(HaveOccurred())

	for _, email := range []string{
		"user@example.com",
		"User@EXAMPLE.COM",
		"user@mail.example.com",
		"user@example.com.",
		`"a@b"@spam.io`,
		"user@fqdn.org",
		"user@example.com@allowed.io",
	} {
		g.Expect(emailBlocked(domains, email)).To(BeTrue(), email)
	}
	for _, email := range []string{
		"user@notexample.com",
		"user@example.com.evil.io",
		"user@example.co",
		"example.com",
		"",
	} {
		g.Expect(emailBlocked(domains, email)).To(BeFalse(), email)
	}
	g.Expect(emailBlocked(nil, "user@example.com")).To(BeFalse())

	for _, invalid := range []string{"example..com", "user@example.com", "exa_mple.com", "-example.com", "example.com/x"} {
		_, err := envparse.EmailDomainList(invalid)
		g.Expect(err).To(HaveOccurred(), invalid)
	}
}

func TestIPBlocked(t *testing.T) {
	g := NewWithT(t)
	prefixes, err := envparse.IPPrefixList("203.0.113.7, 198.51.100.1/24,2001:db8::/32,::ffff:192.0.2.0/120")
	g.Expect(err).NotTo(HaveOccurred())

	for _, ip := range []string{
		"203.0.113.7", "::ffff:203.0.113.7", "198.51.100.200", "2001:db8::1", "192.0.2.9", "::ffff:192.0.2.9",
	} {
		g.Expect(ipBlocked(prefixes, netip.MustParseAddr(ip))).To(BeTrue(), ip)
	}
	for _, ip := range []string{"203.0.113.8", "198.51.101.1", "2001:db9::1", "192.0.3.1"} {
		g.Expect(ipBlocked(prefixes, netip.MustParseAddr(ip))).To(BeFalse(), ip)
	}
	g.Expect(ipBlocked(prefixes, netip.Addr{})).To(BeFalse())

	for _, invalid := range []string{"203.0.113", "198.51.100.0/33", "example.com"} {
		_, err := envparse.IPPrefixList(invalid)
		g.Expect(err).To(HaveOccurred(), invalid)
	}
}

func TestRequestBlocked(t *testing.T) {
	g := NewWithT(t)
	prefixes, err := envparse.IPPrefixList("203.0.113.0/24")
	g.Expect(err).NotTo(HaveOccurred())

	request := func(remoteAddr string, xff ...string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = remoteAddr
		for _, value := range xff {
			r.Header.Add("X-Forwarded-For", value)
		}
		return r
	}

	g.Expect(requestBlocked(prefixes, request("203.0.113.7:1234"))).To(BeTrue())
	g.Expect(requestBlocked(prefixes, request("203.0.113.7:1234", "198.51.100.1"))).To(BeTrue())
	g.Expect(requestBlocked(prefixes, request("10.0.0.1:1234", "198.51.100.1, 203.0.113.7"))).To(BeTrue())
	g.Expect(requestBlocked(prefixes, request("10.0.0.1:1234", "203.0.113.7", "198.51.100.1"))).To(BeTrue())
	g.Expect(requestBlocked(prefixes, request("10.0.0.1:1234", "garbage, 203.0.113.7"))).To(BeTrue())
	g.Expect(requestBlocked(prefixes, request("10.0.0.1:1234", "198.51.100.1"))).To(BeFalse())
	g.Expect(requestBlocked(nil, request("203.0.113.7:1234"))).To(BeFalse())
}
