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
		before := subscription.GetLogQueryWindowStart(types.SubscriptionTypeCommunity)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, time.Time{})
		after := subscription.GetLogQueryWindowStart(types.SubscriptionTypeCommunity)
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

	t.Run("after slightly older than the exact window is allowed (start-of-day slack)", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - time.Minute)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeCommunity, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))
	})

	t.Run("after older than the window including the slack is rejected", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowCommunity - 25*time.Hour)
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

	t.Run("business subscriptions get the 30-day window", func(t *testing.T) {
		g := NewWithT(t)
		requested := time.Now().Add(-subscription.LogQueryWindowDefault - time.Minute)
		resolved, err := resolveLogQueryStart(types.SubscriptionTypeBusiness, requested)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resolved).To(Equal(requested))

		tooOld := time.Now().Add(-subscription.LogQueryWindowBusiness - 25*time.Hour)
		_, err = resolveLogQueryStart(types.SubscriptionTypeBusiness, tooOld)
		g.Expect(err).To(MatchError(apierrors.ErrBadRequest))
	})
}
