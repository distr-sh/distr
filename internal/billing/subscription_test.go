package billing

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v86"
)

func subscriptionWithLookupKeys(keyQuantities map[string]int64) stripe.Subscription {
	items := make([]*stripe.SubscriptionItem, 0, len(keyQuantities))
	for key, qty := range keyQuantities {
		items = append(items, &stripe.SubscriptionItem{
			Price:    &stripe.Price{LookupKey: key},
			Quantity: qty,
		})
	}
	return stripe.Subscription{Items: &stripe.SubscriptionItemList{Data: items}}
}

func TestGetSubscriptionType(t *testing.T) {
	t.Run("pro price keys map to pro", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithLookupKeys(map[string]int64{
			PriceKeyProCustomerMonthly: 5,
			PriceKeyProUserMonthly:     3,
		})
		result, err := GetSubscriptionType(sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(*result).To(Equal(types.SubscriptionTypePro))
	})

	t.Run("business price keys map to business", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithLookupKeys(map[string]int64{
			PriceKeyBusinessCustomerYearly: 5,
			PriceKeyBusinessUserYearly:     3,
		})
		result, err := GetSubscriptionType(sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(*result).To(Equal(types.SubscriptionTypeBusiness))
	})

	t.Run("mixed pro and business price keys are rejected", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithLookupKeys(map[string]int64{
			PriceKeyProCustomerMonthly:  5,
			PriceKeyBusinessUserMonthly: 3,
		})
		_, err := GetSubscriptionType(sub)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("unknown price keys are rejected", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithLookupKeys(map[string]int64{"distr_starter_customer_monthly": 5})
		_, err := GetSubscriptionType(sub)
		g.Expect(err).To(HaveOccurred())
	})
}

func TestGetQuantitiesAndPeriodWithBusinessKeys(t *testing.T) {
	g := NewWithT(t)
	sub := subscriptionWithLookupKeys(map[string]int64{
		PriceKeyBusinessCustomerMonthly: 7,
		PriceKeyBusinessUserMonthly:     2,
	})

	customerQty, err := GetCustomerOrganizationQty(sub)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(customerQty.Value()).To(Equal(int64(7)))

	userQty, err := GetUserAccountQty(sub)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(userQty.Value()).To(Equal(int64(2)))

	period, err := GetSubscriptionPeriod(sub)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(period).To(Equal(types.SubscriptionPeriodMonthly))

	yearlySub := subscriptionWithLookupKeys(map[string]int64{
		PriceKeyBusinessCustomerYearly: 7,
		PriceKeyBusinessUserYearly:     2,
	})
	period, err = GetSubscriptionPeriod(yearlySub)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(period).To(Equal(types.SubscriptionPeriodYearly))
}
