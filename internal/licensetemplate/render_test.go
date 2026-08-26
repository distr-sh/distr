package licensetemplate

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v86"
)

func subscriptionWithItems(items ...*stripe.SubscriptionItem) stripe.Subscription {
	return stripe.Subscription{
		Items: &stripe.SubscriptionItemList{Data: items},
	}
}

func subscriptionItem(lookupKey string, quantity int64) *stripe.SubscriptionItem {
	return &stripe.SubscriptionItem{
		Price:    &stripe.Price{LookupKey: lookupKey},
		Quantity: quantity,
	}
}

func tmpl(payloadTemplate string) types.LicenseTemplate {
	return types.LicenseTemplate{PayloadTemplate: payloadTemplate}
}

func TestRenderPayload(t *testing.T) {
	t.Run("static JSON is rendered as-is", func(t *testing.T) {
		g := NewWithT(t)
		result, err := RenderPayload(tmpl(`{"plan":"pro"}`), stripe.Subscription{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"plan":"pro"}`))
	})

	t.Run("invalid template syntax returns error", func(t *testing.T) {
		g := NewWithT(t)
		_, err := RenderPayload(tmpl(`{"plan": "{{ .Unclosed }`), stripe.Subscription{})
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("rendered output that is not valid JSON returns error", func(t *testing.T) {
		g := NewWithT(t)
		_, err := RenderPayload(tmpl(`not json`), stripe.Subscription{})
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("hasItem returns true when subscription contains the lookup key", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithItems(subscriptionItem("pro-monthly", 1))
		result, err := RenderPayload(tmpl(`{"isPro": {{ hasItem "pro-monthly" }}}`), sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"isPro": true}`))
	})

	t.Run("hasItem returns false when subscription does not contain the lookup key", func(t *testing.T) {
		g := NewWithT(t)
		result, err := RenderPayload(tmpl(`{"isPro": {{ hasItem "pro-monthly" }}}`), stripe.Subscription{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"isPro": false}`))
	})

	t.Run("hasItem matches any of multiple lookup keys", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithItems(subscriptionItem("pro-yearly", 1))
		result, err := RenderPayload(tmpl(`{"isPro": {{ hasItem "pro-monthly" "pro-yearly" }}}`), sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"isPro": true}`))
	})

	t.Run("itemQuantity returns quantity for matching lookup key", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithItems(subscriptionItem("seats-monthly", 5))
		result, err := RenderPayload(tmpl(`{"seats": {{ itemQuantity "seats-monthly" }}}`), sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"seats": 5}`))
	})

	t.Run("itemQuantity returns 0 when lookup key is not present", func(t *testing.T) {
		g := NewWithT(t)
		result, err := RenderPayload(tmpl(`{"seats": {{ itemQuantity "seats-monthly" }}}`), stripe.Subscription{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"seats": 0}`))
	})

	t.Run("hasItem and itemQuantity can be combined in one template", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithItems(
			subscriptionItem("business-monthly", 1),
			subscriptionItem("seats-monthly", 3),
		)
		result, err := RenderPayload(
			tmpl(
				`{"plan": "{{ if hasItem "business-monthly" }}business{{ else }}pro{{ end }}", `+
					`"seats": {{ itemQuantity "seats-monthly" }}}`,
			),
			sub,
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(`{"plan": "business", "seats": 3}`))
	})
}

//nolint:lll // kept verbatim so it can be pasted into the license template form unchanged
const distrLicenseTemplate = `{
  "ld": {
    "enf": true,
    "t": "{{ if hasItem "distr_business_customer_monthly" "distr_business_customer_yearly" }}business{{ else }}pro{{ end }}",
    "p": "{{ if hasItem "distr_pro_customer_yearly" "distr_business_customer_yearly" }}yearly{{ else }}monthly{{ end }}",
    "mo": 1,
    "mou": {{ or (itemQuantity "distr_pro_user_monthly") (itemQuantity "distr_pro_user_yearly") (itemQuantity "distr_business_user_monthly") (itemQuantity "distr_business_user_yearly") }},
    "moc": {{ or (itemQuantity "distr_pro_customer_monthly") (itemQuantity "distr_pro_customer_yearly") (itemQuantity "distr_business_customer_monthly") (itemQuantity "distr_business_customer_yearly") }}
  }
}`

func TestRenderPayload_DistrLicenseTemplate(t *testing.T) {
	t.Run("pro monthly subscription", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithItems(
			subscriptionItem("distr_pro_customer_monthly", 5),
			subscriptionItem("distr_pro_user_monthly", 3),
		)
		result, err := RenderPayload(tmpl(distrLicenseTemplate), sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(
			`{"ld":{"enf":true,"t":"pro","p":"monthly","mo":1,"mou":3,"moc":5}}`))
	})

	t.Run("business yearly subscription", func(t *testing.T) {
		g := NewWithT(t)
		sub := subscriptionWithItems(
			subscriptionItem("distr_business_customer_yearly", 20),
			subscriptionItem("distr_business_user_yearly", 7),
		)
		result, err := RenderPayload(tmpl(distrLicenseTemplate), sub)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(
			`{"ld":{"enf":true,"t":"business","p":"yearly","mo":1,"mou":7,"moc":20}}`))
	})

	t.Run("subscription without any known price key", func(t *testing.T) {
		g := NewWithT(t)
		result, err := RenderPayload(tmpl(distrLicenseTemplate), stripe.Subscription{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(result)).To(MatchJSON(
			`{"ld":{"enf":true,"t":"pro","p":"monthly","mo":1,"mou":0,"moc":0}}`))
	})
}
