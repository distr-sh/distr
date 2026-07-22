package handlers

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestResolveLogQueryStart(t *testing.T) {
	t.Run("zero after defaults to the window start", func(t *testing.T) {
		g := NewWithT(t)
		before := time.Now().Add(-subscription.LogQueryWindowCommunity)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, time.Time{})
		after := time.Now().Add(-subscription.LogQueryWindowCommunity)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(And(
			BeTemporally(">=", before),
			BeTemporally("<=", after),
		))
	})

	t.Run("after within the window is returned unchanged", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-time.Hour)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))
	})

	t.Run("after older than the window is rejected", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - time.Minute)
		_, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, requested)
		g.Expect(err).To(MatchError(apierrors.ErrBadRequest))
	})

	t.Run("pro subscriptions get the larger default window", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - time.Minute)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypePro, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))
	})
}
