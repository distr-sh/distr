package customdomains

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

// A domain is dropped on evidence that its record is wrong, and never because verification merely
// stopped happening: an inconclusive check leaves a working domain in place, so a resolver outage
// cannot move every organization off its own hostname at once.
func TestVerified(t *testing.T) {
	longAgo := time.Now().Add(-30 * 24 * time.Hour)
	recently := time.Now().Add(-time.Minute)

	t.Run("a domain that was verified long ago and has no error is still used", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(types.CustomDomain{VerifiedAt: &longAgo, VerificationCheckedAt: &recently}.Verified()).To(BeTrue())
	})

	t.Run("a domain whose record was found to be wrong is dropped", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(types.CustomDomain{
			VerifiedAt:            &longAgo,
			VerificationCheckedAt: &recently,
			VerificationError:     new("no CNAME record found"),
		}.Verified()).To(BeFalse())
	})

	t.Run("a domain that has never been verified is not used", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(types.CustomDomain{}.Verified()).To(BeFalse())
	})
}
