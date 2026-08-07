package dns

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestCompareCNAME(t *testing.T) {
	t.Run("matches ignoring the trailing dot LookupCNAME appends", func(t *testing.T) {
		g := NewWithT(t)
		verified, _ := compareCNAME("custom-app.distr.sh.", "custom-app.distr.sh")
		g.Expect(verified).To(BeTrue())
	})

	t.Run("matches ignoring case", func(t *testing.T) {
		g := NewWithT(t)
		verified, _ := compareCNAME("Custom-App.Distr.sh.", "custom-app.distr.sh")
		g.Expect(verified).To(BeTrue())
	})

	t.Run("reports a mismatch with both hosts in the detail", func(t *testing.T) {
		g := NewWithT(t)
		verified, detail := compareCNAME("elsewhere.example.com.", "custom-app.distr.sh")
		g.Expect(verified).To(BeFalse())
		g.Expect(detail).To(ContainSubstring("elsewhere.example.com"))
		g.Expect(detail).To(ContainSubstring("custom-app.distr.sh"))
	})

	t.Run("no CNAME record resolves to the domain itself, reported as a mismatch", func(t *testing.T) {
		g := NewWithT(t)
		verified, _ := compareCNAME("acme.example.com.", "custom-app.distr.sh")
		g.Expect(verified).To(BeFalse())
	})
}
