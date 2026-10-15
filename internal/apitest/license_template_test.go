package apitest_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

const (
	seatsTemplate = `{"seats": {{ itemQuantity "seats" }}, "support": {{ hasItem "support" "support_yearly" }}}`
	webhookSecret = "whsec_integrationtest"
)

type licenseTemplate struct {
	ID                        uuid.UUID `json:"id"`
	Name                      string    `json:"name"`
	PayloadTemplate           string    `json:"payloadTemplate"`
	ExpirationGracePeriodDays int       `json:"expirationGracePeriodDays"`
}

func billingVendor(h *apitest.Harness) *apitest.User {
	vendor := h.RegisterVendor()
	vendor.GrantFeature(types.FeatureVendorBilling)
	return vendor
}

func templateBody(name, payloadTemplate string, graceDays int) api.CreateLicenseTemplateRequest {
	return api.CreateLicenseTemplateRequest{
		Name: name, PayloadTemplate: payloadTemplate, ExpirationGracePeriodDays: graceDays,
	}
}

func createLicenseTemplate(vendor *apitest.User, name string, graceDays int) uuid.UUID {
	return apitest.Expect[apitest.Entity](
		vendor.Post(apitest.LicenseTemplates, templateBody(name, seatsTemplate, graceDays)), http.StatusOK).ID
}

func templatedLicenseKey(vendor *apitest.User, name string, customer *uuid.UUID, template uuid.UUID) uuid.UUID {
	body := object{"name": name, "licenseTemplateId": template}
	if customer != nil {
		body["customerOrganizationId"] = *customer
	}
	return apitest.Expect[apitest.Entity](vendor.Post(apitest.LicenseKeys, body), http.StatusOK).ID
}

func revisionCount(h *apitest.Harness, key uuid.UUID) int {
	return apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseKeyRevision WHERE license_key_id = $1`, key)
}

func latestRevisionPayload(h *apitest.Harness, key uuid.UUID) string {
	return apitest.QueryValue[string](h, `SELECT payload::text FROM LicenseKeyRevision
		WHERE license_key_id = $1 ORDER BY created_at DESC LIMIT 1`, key)
}

func latestRevisionExpiry(h *apitest.Harness, key uuid.UUID) time.Time {
	return apitest.QueryValue[time.Time](h, `SELECT expires_at FROM LicenseKeyRevision
		WHERE license_key_id = $1 ORDER BY created_at DESC LIMIT 1`, key)
}

func subscription(status stripe.SubscriptionStatus, licenseKey any, items ...object) object {
	return object{
		"id":       "sub_integrationtest",
		"object":   "subscription",
		"status":   status,
		"metadata": object{"licenseKeyId": licenseKey},
		"items":    object{"object": "list", "data": items},
	}
}

func subscriptionItem(lookupKey string, quantity int, periodEnd time.Time) object {
	return object{
		"id":                 "si_" + lookupKey,
		"object":             "subscription_item",
		"quantity":           quantity,
		"current_period_end": periodEnd.Unix(),
		"price":              object{"id": "price_" + lookupKey, "object": "price", "lookup_key": lookupKey},
	}
}

// stripeEvent is a webhook delivery of eventType as Stripe signs it with secret at timestamp.
type stripeEvent struct {
	eventType stripe.EventType
	object    object
	secret    string
	timestamp time.Time
}

func (e stripeEvent) send(h *apitest.Harness, organization string) *apitest.Response {
	payload, err := json.Marshal(object{
		"id":          "evt_" + uuid.NewString(),
		"object":      "event",
		"api_version": stripe.APIVersion,
		"type":        e.eventType,
		"data":        object{"object": e.object},
	})
	if err != nil {
		panic(err)
	}
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload: payload, Secret: e.secret, Timestamp: e.timestamp,
	})
	// Do encodes a json.RawMessage verbatim, so the body is the payload that was signed.
	return h.Anonymous().WithHeader("Stripe-Signature", signed.Header).
		Post("/api/v1/webhook/"+organization+"/stripe", json.RawMessage(payload))
}

func subscriptionUpdated(object object) stripeEvent {
	return stripeEvent{eventType: stripe.EventTypeCustomerSubscriptionUpdated, object: object, secret: webhookSecret}
}

func setWebhookSecret(vendor *apitest.User, secret string) {
	vendor.Put(apitest.OrganizationWebhook, api.UpdateOrganizationWebhookRequest{WebhookSecret: &secret}).
		ExpectStatus(http.StatusNoContent)
}

func TestLicenseTemplateLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	customer := vendor.CreateCustomerOrganization("Customer")
	g := NewWithT(t)

	second := createLicenseTemplate(vendor, "b template", 0)
	first := apitest.Expect[licenseTemplate](vendor.Post(apitest.LicenseTemplates,
		templateBody("a template", `{"tier": "gold"}`, 7)), http.StatusOK)
	g.Expect(first).To(Equal(licenseTemplate{
		ID: first.ID, Name: "a template", PayloadTemplate: `{"tier": "gold"}`, ExpirationGracePeriodDays: 7,
	}))
	g.Expect(apitest.IDs(vendor.Get(apitest.LicenseTemplates), http.StatusOK)).
		To(Equal([]uuid.UUID{first.ID, second}), "templates are listed by name")

	updated := apitest.Expect[licenseTemplate](vendor.Put(apitest.Path(apitest.LicenseTemplates, first.ID),
		api.UpdateLicenseTemplateRequest{Name: "c template", PayloadTemplate: seatsTemplate, ExpirationGracePeriodDays: 1}),
		http.StatusOK)
	g.Expect(updated).To(Equal(licenseTemplate{
		ID: first.ID, Name: "c template", PayloadTemplate: seatsTemplate, ExpirationGracePeriodDays: 1,
	}))
	g.Expect(apitest.Expect[[]licenseTemplate](vendor.Get(apitest.LicenseTemplates), http.StatusOK)).
		To(Equal([]licenseTemplate{
			{ID: second, Name: "b template", PayloadTemplate: seatsTemplate},
			updated,
		}))

	other := billingVendor(h)
	g.Expect(apitest.IDs(other.Get(apitest.LicenseTemplates), http.StatusOK)).To(BeEmpty())
	createLicenseTemplate(other, "b template", 0)

	key := templatedLicenseKey(vendor, "key", &customer, second)
	g.Expect(apitest.QueryValue[*uuid.UUID](h, `SELECT license_template_id FROM LicenseKey WHERE id = $1`, key)).
		To(HaveValue(Equal(second)))
	g.Expect(revisionCount(h, key)).To(BeZero(), "a templated key waits for the first subscription event")
	vendor.Get(apitest.Path(apitest.LicenseKeys, key) + "/token").ExpectStatus(http.StatusUnprocessableEntity)

	vendor.Delete(apitest.Path(apitest.LicenseTemplates, second)).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.IDs(vendor.Get(apitest.LicenseTemplates), http.StatusOK)).To(Equal([]uuid.UUID{first.ID}))
	g.Expect(apitest.QueryValue[*uuid.UUID](h, `SELECT license_template_id FROM LicenseKey WHERE id = $1`, key)).
		To(BeNil(), "deleting a template detaches the keys rendered from it")
}

func TestLicenseTemplateErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	existing := createLicenseTemplate(vendor, "existing", 2)
	createLicenseTemplate(vendor, "taken", 0)
	existingPath := apitest.Path(apitest.LicenseTemplates, existing)
	unknownPath := apitest.Path(apitest.LicenseTemplates, uuid.New())
	g := NewWithT(t)

	create := func(name string, body any) request {
		return request{name, http.MethodPost, apitest.LicenseTemplates, body}
	}
	put := func(name string, body any) request { return request{name, http.MethodPut, existingPath, body} }
	g.Expect(statuses(vendor.Client, []request{
		create("no name", templateBody("", seatsTemplate, 0)),
		create("no payload template", templateBody("x", "", 0)),
		create("negative grace period", templateBody("x", seatsTemplate, -1)),
		create("unclosed action", templateBody("x", `{"seats": {{ itemQuantity "seats" }`, 0)),
		create("unknown function", templateBody("x", `{"seats": {{ seats }}}`, 0)),
		create("unknown subscription field", templateBody("x", `{"seats": {{ .Seats }}}`, 0)),
		create("rendering invalid JSON", templateBody("x", `{"seats": {{ itemQuantity "seats" }},}`, 0)),
		create("duplicate name", templateBody("existing", seatsTemplate, 0)),
		create("malformed body", "{"),
		put("put without name", templateBody("", seatsTemplate, 0)),
		put("put negative grace period", templateBody("existing", seatsTemplate, -1)),
		put("put rendering invalid JSON", templateBody("existing", `{"seats": }`, 0)),
		put("put to a taken name", templateBody("taken", seatsTemplate, 0)),
		put("put malformed body", "{"),
		{
			"put malformed id", http.MethodPut, apitest.LicenseTemplates + "/nope",
			templateBody("x", seatsTemplate, 0),
		},
		{"put unknown", http.MethodPut, unknownPath, templateBody("x", seatsTemplate, 0)},
		{"delete malformed id", http.MethodDelete, apitest.LicenseTemplates + "/nope", nil},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"no name":                    http.StatusBadRequest,
		"no payload template":        http.StatusBadRequest,
		"negative grace period":      http.StatusBadRequest,
		"unclosed action":            http.StatusBadRequest,
		"unknown function":           http.StatusBadRequest,
		"unknown subscription field": http.StatusBadRequest,
		"rendering invalid JSON":     http.StatusBadRequest,
		"duplicate name":             http.StatusBadRequest,
		"malformed body":             http.StatusBadRequest,
		"put without name":           http.StatusBadRequest,
		"put negative grace period":  http.StatusBadRequest,
		"put rendering invalid JSON": http.StatusBadRequest,
		"put to a taken name":        http.StatusBadRequest,
		"put malformed body":         http.StatusBadRequest,
		"put malformed id":           http.StatusBadRequest,
		"put unknown":                http.StatusNotFound,
		"delete malformed id":        http.StatusBadRequest,
		"delete unknown":             http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseTemplate`)).To(Equal(2))
	g.Expect(apitest.QueryValue[bool](h, `SELECT name = 'existing' AND payload_template = $2
		AND expiration_grace_period_days = 2 FROM LicenseTemplate WHERE id = $1`, existing, seatsTemplate)).
		To(BeTrue())
}

func TestLicenseTemplateRejectsPayloadsALicenseKeyRejects(t *testing.T) {
	knownBug(t, "a license template is only checked to render valid JSON, while a license key payload has "+
		"to be an object without registered claims. A template rendering an array, a number or a string is "+
		"stored as the revision of every key using it, after which the token of the key answers 500")
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		{"array", http.MethodPost, apitest.LicenseTemplates, templateBody("a", `[1]`, 0)},
		{"number", http.MethodPost, apitest.LicenseTemplates, templateBody("b", `{{ itemQuantity "seats" }}`, 0)},
		{"string", http.MethodPost, apitest.LicenseTemplates, templateBody("c", `"gold"`, 0)},
		{"exp claim", http.MethodPost, apitest.LicenseTemplates, templateBody("d", `{"exp": 1}`, 0)},
		{"org claim", http.MethodPost, apitest.LicenseTemplates, templateBody("e", `{"org": "x"}`, 0)},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseTemplate`)).To(BeZero())
}

func TestLicenseTemplateAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	customer := vendor.CreateCustomerOrganization("Customer")
	template := createLicenseTemplate(vendor, "template", 0)
	templatePath := apitest.Path(apitest.LicenseTemplates, template)
	g := NewWithT(t)

	routes := []request{
		{"list", http.MethodGet, apitest.LicenseTemplates, nil},
		{"create", http.MethodPost, apitest.LicenseTemplates, templateBody("new", seatsTemplate, 0)},
		{"update", http.MethodPut, templatePath, templateBody("renamed", seatsTemplate, 0)},
		{"delete", http.MethodDelete, templatePath, nil},
		{"get webhook", http.MethodGet, apitest.OrganizationWebhook, nil},
		{
			"set webhook", http.MethodPut, apitest.OrganizationWebhook,
			api.UpdateOrganizationWebhookRequest{WebhookSecret: new(webhookSecret)},
		},
	}
	g.Expect(statuses(h.RegisterVendor().Client, routes)).
		To(Equal(allWithStatus(routes, http.StatusForbidden)), "vendor billing is granted per organization")
	g.Expect(statuses(vendor.Invite(types.UserRoleAdmin, &customer).Client, routes)).
		To(Equal(allWithStatus(routes, http.StatusForbidden)))
	g.Expect(statuses(vendor.Invite(types.UserRoleReadOnly, nil).Client, routes)).To(Equal(map[string]int{
		"list":        http.StatusOK,
		"create":      http.StatusForbidden,
		"update":      http.StatusForbidden,
		"delete":      http.StatusForbidden,
		"get webhook": http.StatusOK,
		"set webhook": http.StatusForbidden,
	}))
	g.Expect(statuses(vendor.Invite(types.UserRoleReadWrite, nil).Client, []request{
		{"create", http.MethodPost, apitest.LicenseTemplates, templateBody("by editor", seatsTemplate, 0)},
		{
			"set webhook", http.MethodPut, apitest.OrganizationWebhook,
			api.UpdateOrganizationWebhookRequest{WebhookSecret: new(webhookSecret)},
		},
	})).To(Equal(map[string]int{"create": http.StatusOK, "set webhook": http.StatusForbidden}),
		"templates take a read-write user, the webhook secret an admin")

	other := billingVendor(h)
	otherTemplate := createLicenseTemplate(other, "foreign", 0)
	g.Expect(statuses(other.Client, []request{
		{"update", http.MethodPut, templatePath, templateBody("renamed", seatsTemplate, 0)},
		{"delete", http.MethodDelete, templatePath, nil},
	})).To(HaveEach(http.StatusNotFound))
	g.Expect(apitest.IDs(other.Get(apitest.LicenseTemplates), http.StatusOK)).To(Equal([]uuid.UUID{otherTemplate}))
	key := templatedLicenseKey(vendor, "key", &customer, template)
	g.Expect(statuses(vendor.Client, []request{
		{
			"key with another vendor's template", http.MethodPost, apitest.LicenseKeys,
			object{"name": "x", "customerOrganizationId": customer, "licenseTemplateId": otherTemplate},
		},
		{
			"key moved to another vendor's template", http.MethodPut, apitest.Path(apitest.LicenseKeys, key),
			object{"licenseTemplateId": otherTemplate},
		},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.QueryValue[uuid.UUID](h, `SELECT license_template_id FROM LicenseKey WHERE id = $1`, key)).
		To(Equal(template))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM LicenseTemplate WHERE id = $1`, template)).
		To(Equal("template"))
}

func TestVendorStripeWebhookSecret(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	g := NewWithT(t)
	configured := func() bool {
		return apitest.Expect[api.OrganizationWebhookResponse](vendor.Get(apitest.OrganizationWebhook),
			http.StatusOK).Configured
	}

	set := func(name, secret string) request {
		return request{
			name, http.MethodPut, apitest.OrganizationWebhook,
			api.UpdateOrganizationWebhookRequest{WebhookSecret: &secret},
		}
	}
	g.Expect(statuses(vendor.Client, []request{
		set("without prefix", "integrationtest"),
		set("prefix only", "whsec_"),
		set("special characters", "whsec_a-b"),
		set("too long", "whsec_"+strings.Repeat("a", 129)),
		{"malformed body", http.MethodPut, apitest.OrganizationWebhook, "{"},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(configured()).To(BeFalse())

	setWebhookSecret(vendor, webhookSecret)
	g.Expect(configured()).To(BeTrue())
	g.Expect(apitest.QueryValue[bool](h, `SELECT stripe_webhook_secret IS NULL AND stripe_webhook_secret_enc IS NOT NULL
		FROM Organization WHERE id = $1`, vendor.OrganizationID)).To(BeTrue(), "the secret is stored encrypted")

	vendor.Put(apitest.OrganizationWebhook, api.UpdateOrganizationWebhookRequest{}).ExpectStatus(http.StatusNoContent)
	g.Expect(configured()).To(BeFalse())
	g.Expect(apitest.QueryValue[bool](h, `SELECT stripe_webhook_secret_enc IS NULL FROM Organization WHERE id = $1`,
		vendor.OrganizationID)).To(BeTrue())
}

func TestVendorStripeWebhookRevisesLicenseKeys(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	h.Exec(`UPDATE UserAccount SET email_verified_at = now() WHERE id = $1`, vendor.ID)
	setWebhookSecret(vendor, webhookSecret)
	customer := vendor.CreateCustomerOrganization("Acme")
	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	vendor.Invite(types.UserRoleReadOnly, &customer)
	vendor.Invite(types.UserRoleReadOnly, nil)
	otherCustomer := vendor.CreateCustomerOrganization("Other")
	vendor.Invite(types.UserRoleAdmin, &otherCustomer)
	template := createLicenseTemplate(vendor, "template", 3)
	key := templatedLicenseKey(vendor, "acme key", &customer, template)
	keyPath := apitest.Path(apitest.LicenseKeys, key)
	mails := mailsAfterSetup(h)
	g := NewWithT(t)
	periodEnd := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second).UTC()
	org := vendor.OrganizationID.String()

	update := subscriptionUpdated(subscription(stripe.SubscriptionStatusActive, key,
		subscriptionItem("seats", 7, periodEnd), subscriptionItem("support_yearly", 1, periodEnd)))
	update.send(h, org).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, key)).To(Equal(1))
	g.Expect(latestRevisionPayload(h, key)).To(MatchJSON(`{"seats": 7, "support": true}`))
	g.Expect(latestRevisionExpiry(h, key)).To(BeTemporally("==", periodEnd.AddDate(0, 0, 3)),
		"a key expires at the end of the paid period plus the grace period of its template")
	vendor.Get(keyPath + "/token").ExpectStatus(http.StatusOK)
	customerAdmin.Get(keyPath + "/token").ExpectStatus(http.StatusOK)

	sent := mails()
	g.Expect(sent).To(HaveLen(2), "only verified admins of the vendor and of the key's customer are told")
	g.Expect(sent).To(ContainElement(And(
		HaveField("To", Equal([]string{vendor.Email})),
		HaveField("Subject", Equal("License key updated for Acme: acme key")),
	)))
	g.Expect(sent).To(ContainElement(And(
		HaveField("To", Equal([]string{customerAdmin.Email})),
		HaveField("Subject", Equal("License key updated: acme key")),
	)))

	update.send(h, org).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, key)).To(Equal(1), "a redelivered event changes nothing")
	g.Expect(mails()).To(HaveLen(2))

	subscriptionUpdated(subscription(stripe.SubscriptionStatusActive, key,
		subscriptionItem("seats", 9, periodEnd))).send(h, org).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, key)).To(Equal(2))
	g.Expect(latestRevisionPayload(h, key)).To(MatchJSON(`{"seats": 9, "support": false}`))

	canceledAt := time.Now().Add(-time.Hour).Truncate(time.Second).UTC()
	canceled := subscription(stripe.SubscriptionStatusCanceled, key, subscriptionItem("seats", 9, periodEnd))
	canceled["canceled_at"] = canceledAt.Unix()
	stripeEvent{eventType: stripe.EventTypeCustomerSubscriptionDeleted, object: canceled, secret: webhookSecret}.
		send(h, org).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, key)).To(Equal(3))
	g.Expect(latestRevisionExpiry(h, key)).To(BeTemporally("==", canceledAt.AddDate(0, 0, 3)),
		"a canceled subscription expires the key a grace period after the cancellation")

	stripeEvent{eventType: stripe.EventTypeInvoicePaid, object: object{"id": "in_1"}, secret: webhookSecret}.
		send(h, org).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, key)).To(Equal(3))
}

func TestVendorStripeWebhookRejectsWhatItCannotAttribute(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := billingVendor(h)
	setWebhookSecret(vendor, webhookSecret)
	customer := vendor.CreateCustomerOrganization("Customer")
	template := createLicenseTemplate(vendor, "template", 0)
	key := templatedLicenseKey(vendor, "key", &customer, template)
	manualKey := apitest.Expect[apitest.Entity](vendor.Post(apitest.LicenseKeys,
		licenseKeyBody("manual", &customer, object{"seats": 1})), http.StatusOK).ID
	detachedTemplate := createLicenseTemplate(vendor, "detached", 0)
	detachedKey := templatedLicenseKey(vendor, "detached", &customer, detachedTemplate)
	vendor.Delete(apitest.Path(apitest.LicenseTemplates, detachedTemplate)).ExpectStatus(http.StatusNoContent)
	other := billingVendor(h)
	const otherSecret = "whsec_othervendor"
	setWebhookSecret(other, otherSecret)
	otherKey := templatedLicenseKey(other, "foreign", nil, createLicenseTemplate(other, "template", 0))
	unconfigured := billingVendor(h)
	mails := mailsAfterSetup(h)
	g := NewWithT(t)
	periodEnd := time.Now().Add(24 * time.Hour)
	org := vendor.OrganizationID.String()

	event := func(licenseKey any) stripeEvent {
		return subscriptionUpdated(subscription(stripe.SubscriptionStatusActive, licenseKey,
			subscriptionItem("seats", 5, periodEnd)))
	}
	withSecret := func(secret string) stripeEvent {
		e := event(key)
		e.secret = secret
		return e
	}
	stale := event(key)
	stale.timestamp = time.Now().Add(-time.Hour)
	unsigned := h.Anonymous().Post("/api/v1/webhook/"+org+"/stripe", json.RawMessage(`{"id": "evt_1"}`))

	g.Expect(map[string]int{
		"malformed organization":      event(key).send(h, "nope").StatusCode,
		"unknown organization":        event(key).send(h, uuid.NewString()).StatusCode,
		"organization without secret": event(key).send(h, unconfigured.OrganizationID.String()).StatusCode,
		"unsigned":                    unsigned.StatusCode,
		"wrong secret":                withSecret("whsec_wrong").send(h, org).StatusCode,
		"another vendor's secret":     withSecret(otherSecret).send(h, org).StatusCode,
		"stale signature":             stale.send(h, org).StatusCode,
		"another vendor's key":        event(otherKey).send(h, org).StatusCode,
		"key without template":        event(manualKey).send(h, org).StatusCode,
		"key of a deleted template":   event(detachedKey).send(h, org).StatusCode,
		"unknown key":                 event(uuid.New()).send(h, org).StatusCode,
		"malformed key":               event("nope").send(h, org).StatusCode,
	}).To(Equal(map[string]int{
		"malformed organization":      http.StatusNotFound,
		"unknown organization":        http.StatusNotFound,
		"organization without secret": http.StatusServiceUnavailable,
		"unsigned":                    http.StatusBadRequest,
		"wrong secret":                http.StatusBadRequest,
		"another vendor's secret":     http.StatusBadRequest,
		"stale signature":             http.StatusBadRequest,
		"another vendor's key":        http.StatusOK,
		"key without template":        http.StatusOK,
		"key of a deleted template":   http.StatusOK,
		"unknown key":                 http.StatusOK,
		"malformed key":               http.StatusOK,
	}), "events Distr cannot attribute are acknowledged, so that Stripe stops redelivering them")
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseKeyRevision WHERE license_key_id <> $1`,
		manualKey)).To(BeZero())
	g.Expect(revisionCount(h, manualKey)).To(Equal(1))
	g.Expect(mails()).To(BeEmpty())

	ownEvent := event(otherKey)
	ownEvent.secret = otherSecret
	ownEvent.send(h, other.OrganizationID.String()).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, otherKey)).To(Equal(1))
}

// Not parallel, since it moves the whole process into another zone.
func TestVendorStripeWebhookOutsideUTC(t *testing.T) {
	knownBug(t, "the vendor Stripe webhook passes times in the local zone of the process (time.Unix, time.Now) "+
		"to TIMESTAMP columns, which pgx writes as wall clock. Outside UTC every revision starts and expires "+
		"off by the UTC offset, and a redelivered event is never recognized as unchanged")
	time.Local = time.FixedZone("UTC+2", 2*60*60)
	t.Cleanup(func() { time.Local = time.UTC })
	h := apitest.New(t)
	vendor := billingVendor(h)
	setWebhookSecret(vendor, webhookSecret)
	key := templatedLicenseKey(vendor, "key", nil, createLicenseTemplate(vendor, "template", 0))
	g := NewWithT(t)
	periodEnd := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	update := subscriptionUpdated(subscription(stripe.SubscriptionStatusActive, key,
		subscriptionItem("seats", 1, periodEnd)))

	update.send(h, vendor.OrganizationID.String()).ExpectStatus(http.StatusOK)
	g.Expect(latestRevisionExpiry(h, key)).To(BeTemporally("==", periodEnd))
	g.Expect(apitest.QueryValue[time.Time](h, `SELECT not_before FROM LicenseKeyRevision WHERE license_key_id = $1`,
		key)).To(BeTemporally("<=", time.Now()), "a fresh revision is valid right away")
	update.send(h, vendor.OrganizationID.String()).ExpectStatus(http.StatusOK)
	g.Expect(revisionCount(h, key)).To(Equal(1))
}
