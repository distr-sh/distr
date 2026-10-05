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
