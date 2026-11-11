package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v87"
)

func TestClassifyStripeSubscription(t *testing.T) {
	t.Run("ending the referenced subscription ends it", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(classifyStripeSubscription(new("sub_old"),
			stripe.Subscription{ID: "sub_old", Status: stripe.SubscriptionStatusCanceled})).
			To(Equal(stripeSubscriptionEnded))
	})

	t.Run("a late end of a previous subscription is ignored", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(classifyStripeSubscription(new("sub_new"),
			stripe.Subscription{ID: "sub_old", Status: stripe.SubscriptionStatusCanceled})).
			To(Equal(stripeSubscriptionIgnored))
	})

	t.Run("a retried end after the reference was cleared is ignored", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(classifyStripeSubscription(nil,
			stripe.Subscription{ID: "sub_old", Status: stripe.SubscriptionStatusIncompleteExpired})).
			To(Equal(stripeSubscriptionIgnored))
	})

	t.Run("a live subscription next to another live one replaces it", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(classifyStripeSubscription(new("sub_old"),
			stripe.Subscription{ID: "sub_new", Status: stripe.SubscriptionStatusActive})).
			To(Equal(stripeSubscriptionReplaced))
	})

	t.Run("a new subscription after the previous one ended is applied", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(classifyStripeSubscription(nil,
			stripe.Subscription{ID: "sub_new", Status: stripe.SubscriptionStatusActive})).
			To(Equal(stripeSubscriptionApplied))
	})
}
