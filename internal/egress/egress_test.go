package egress

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

func TestTransportRefusesNonPublicAddress(t *testing.T) {
	g := NewWithT(t)
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer internal.Close()

	_, err := (&http.Client{Transport: Transport(false)}).Get(internal.URL)
	g.Expect(err).To(MatchError(ContainSubstring("non-public address 127.0.0.1")))

	resp, err := (&http.Client{Transport: Transport(true)}).Get(internal.URL)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resp.Body.Close()).To(Succeed())
}

func TestCheckPublicHost(t *testing.T) {
	g := NewWithT(t)
	for _, host := range []string{
		"0.0.0.1",
		"100.64.0.1",
		"127.0.0.2",
		"169.254.169.254",
		"192.31.196.1",
		"198.18.0.1",
		"255.255.255.255",
		"::",
		"::1",
		"::7f00:1",
		"::ffff:127.0.0.1",
		"64:ff9b::a9fe:a9fe",
		"fc00::1",
		"fec0::1",
		"fe80::1%eth0",
		"ff02::1",
		"2001:db8::1",
		"2002:7f00:1::1",
	} {
		g.Expect(CheckPublicHost(t.Context(), host)).To(MatchError(ContainSubstring("non-public address")), host)
	}
	for _, host := range []string{"8.8.8.8", "::ffff:8.8.8.8", "2606:4700:4700::1111"} {
		g.Expect(CheckPublicHost(t.Context(), host)).To(Succeed(), host)
	}
}
