package dns

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestCompareCNAME(t *testing.T) {
	t.Run("matches ignoring the trailing dot LookupCNAME appends", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(compareCNAME("acme.example.com", "whitelabel.distr.sh.", "whitelabel.distr.sh")).To(Succeed())
	})

	t.Run("matches ignoring case", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(compareCNAME("acme.example.com", "WhiteLabel.Distr.sh.", "whitelabel.distr.sh")).To(Succeed())
	})

	t.Run("reports a mismatch with both hosts", func(t *testing.T) {
		g := NewWithT(t)
		err := compareCNAME("acme.example.com", "elsewhere.example.com.", "whitelabel.distr.sh")
		g.Expect(err).To(MatchError(&CNAMEError{
			Domain:   "acme.example.com",
			Resolved: "elsewhere.example.com",
			Expected: "whitelabel.distr.sh",
		}))
		g.Expect(err.Error()).To(Equal("CNAME points to elsewhere.example.com instead of whitelabel.distr.sh"))
	})

	t.Run("no CNAME record resolves to the domain itself, reported as a missing record", func(t *testing.T) {
		g := NewWithT(t)
		err := compareCNAME("Acme.example.com", "acme.example.com.", "whitelabel.distr.sh")
		g.Expect(err).To(MatchError(&CNAMEError{Domain: "acme.example.com", Expected: "whitelabel.distr.sh"}))
		g.Expect(err.Error()).
			To(Equal("no CNAME record found for acme.example.com, it must point to whitelabel.distr.sh"))
	})
}
