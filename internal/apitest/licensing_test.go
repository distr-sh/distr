package apitest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	. "github.com/onsi/gomega"
)

// The licensing endpoints still take and answer internal types, so their bodies are written as JSON
// objects here to keep the tests independent of those structs.
type object = map[string]any

const (
	licenseKeyPublicKey  = "/api/public/v1/license-keys/public-key"
	sqlErrorFragment     = "SQLSTATE"
	foreignKeyConstraint = "foreign key"
)

func entitlementBody(name string, application uuid.UUID, customer *uuid.UUID, versions ...uuid.UUID) object {
	body := object{"name": name, "applicationId": application, "versions": versionRefs(versions)}
	if customer != nil {
		body["customerOrganizationId"] = *customer
	}
	return body
}

func versionRefs(ids []uuid.UUID) []object {
	refs := make([]object, len(ids))
	for i, id := range ids {
		refs[i] = object{"id": id}
	}
	return refs
}

func licenseKeyBody(name string, customer *uuid.UUID, payload any) object {
	body := object{
		"name":      name,
		"notBefore": time.Now().Add(-time.Hour).UTC(),
		"expiresAt": time.Now().Add(24 * time.Hour).UTC(),
		"payload":   payload,
	}
	if customer != nil {
		body["customerOrganizationId"] = *customer
	}
	return body
}

func TestApplicationEntitlementErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	otherApp := vendor.CreateApplication("other app")
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		entitlementBody("existing", app, &customer)), http.StatusOK).ID
	entitlementPath := apitest.Path(apitest.ApplicationEntitlements, entitlement)
	unknownPath := apitest.Path(apitest.ApplicationEntitlements, uuid.New())
	g := NewWithT(t)

	create := func(name string, body any) request {
		return request{name, http.MethodPost, apitest.ApplicationEntitlements, body}
	}
	withRegistry := func(fields object) object {
		body := entitlementBody("registry", app, &customer)
		for key, value := range fields {
			body[key] = value
		}
		return body
	}
	put := func(name string, body object) request {
		return request{name, http.MethodPut, entitlementPath, body}
	}
	withID := func(body object) object {
		body["id"] = entitlement
		return body
	}
	g.Expect(statuses(vendor.Client, []request{
		create("no name", entitlementBody("", app, &customer)),
		create("no application", object{"name": "x", "customerOrganizationId": customer}),
		create("registry URL without credentials", withRegistry(object{"registryUrl": "ghcr.io"})),
		create("registry credentials without URL",
			withRegistry(object{"registryUsername": "user", "registryPassword": "password"})),
		create("duplicate name for the same customer", entitlementBody("existing", app, &customer)),
		create("malformed body", "{"),
		{"get malformed id", http.MethodGet, apitest.ApplicationEntitlements + "/nope", nil},
		{"get unknown", http.MethodGet, unknownPath, nil},
		{
			"list with malformed application filter", http.MethodGet,
			apitest.ApplicationEntitlements + "?applicationId=nope", nil,
		},
		put("put without name", entitlementBody("", app, &customer)),
		put("put to another id", object{"id": uuid.New(), "name": "existing", "applicationId": app}),
		put("put with another application", withID(entitlementBody("existing", otherApp, &customer))),
		put("put with another customer", withID(entitlementBody("existing", app, &otherCustomer))),
		put("put without customer", withID(entitlementBody("existing", app, nil))),
		{"put malformed body", http.MethodPut, entitlementPath, "{"},
		{"put unknown", http.MethodPut, unknownPath, entitlementBody("x", app, &customer)},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"no name":                                http.StatusBadRequest,
		"no application":                         http.StatusBadRequest,
		"registry URL without credentials":       http.StatusBadRequest,
		"registry credentials without URL":       http.StatusBadRequest,
		"duplicate name for the same customer":   http.StatusBadRequest,
		"malformed body":                         http.StatusBadRequest,
		"get malformed id":                       http.StatusBadRequest,
		"get unknown":                            http.StatusNotFound,
		"list with malformed application filter": http.StatusBadRequest,
		"put without name":                       http.StatusBadRequest,
		"put to another id":                      http.StatusBadRequest,
		"put with another application":           http.StatusBadRequest,
		"put with another customer":              http.StatusBadRequest,
		"put without customer":                   http.StatusBadRequest,
		"put malformed body":                     http.StatusBadRequest,
		"put unknown":                            http.StatusNotFound,
		"delete unknown":                         http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ApplicationEntitlement`)).To(Equal(1))
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT name = 'existing' AND application_id = $2 AND customer_organization_id = $3
		FROM ApplicationEntitlement WHERE id = $1`,
		entitlement, app, customer)).To(BeTrue())

	vendor.Post(apitest.ApplicationEntitlements, entitlementBody("existing", app, &otherCustomer)).
		ExpectStatus(http.StatusOK)
}

func TestApplicationEntitlementOwnerAndApplicationCannotChange(t *testing.T) {
	knownBug(t, "updateApplicationEntitlement only checks the owner and the application when the body "+
		"carries the id, because the checks are else-if branches of filling in a missing id")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	entitlementPath := apitest.Path(apitest.ApplicationEntitlements, apitest.Expect[apitest.Entity](
		vendor.Post(apitest.ApplicationEntitlements, entitlementBody("existing", app, &customer)), http.StatusOK).ID)
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		{
			"another application", http.MethodPut, entitlementPath,
			entitlementBody("existing", vendor.CreateApplication("other"), &customer),
		},
		{"another customer", http.MethodPut, entitlementPath, entitlementBody("existing", app, &otherCustomer)},
		{"no customer", http.MethodPut, entitlementPath, entitlementBody("existing", app, nil)},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.QueryValue[uuid.UUID](h, `SELECT customer_organization_id FROM ApplicationEntitlement`)).
		To(Equal(customer))
}

func TestLicensingReferencesOnlyResourcesOfTheOwnOrganization(t *testing.T) {
	knownBug(t, "creating or updating an application entitlement or a license key does not check that "+
		"the application, versions and customer belong to the organization, so a vendor can reference another "+
		"vendor's and then read their names through the entitlement and its customer's application list")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	other := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	customer := vendor.CreateCustomerOrganization("Customer")
	foreignApp := other.CreateApplication("foreign app")
	foreignVersion := other.CreateApplicationVersion(foreignApp, "1.0.0")
	foreignCustomer := other.CreateCustomerOrganization("Foreign Customer")
	entitlementPath := apitest.Path(apitest.ApplicationEntitlements, apitest.Expect[apitest.Entity](
		vendor.Post(apitest.ApplicationEntitlements, entitlementBody("existing", app, &customer)), http.StatusOK).ID)
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		{
			"entitlement for another vendor's application", http.MethodPost, apitest.ApplicationEntitlements,
			entitlementBody("a", foreignApp, &customer),
		},
		{
			"entitlement with another vendor's version", http.MethodPost, apitest.ApplicationEntitlements,
			entitlementBody("b", app, &customer, foreignVersion),
		},
		{
			"entitlement for another vendor's customer", http.MethodPost, apitest.ApplicationEntitlements,
			entitlementBody("c", app, &foreignCustomer),
		},
		{
			"entitlement updated with another vendor's version", http.MethodPut, entitlementPath,
			entitlementBody("existing", app, &customer, foreignVersion),
		},
		{
			"license key for another vendor's customer", http.MethodPost, apitest.LicenseKeys,
			licenseKeyBody("key", &foreignCustomer, object{}),
		},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.IDs(vendor.Invite(types.UserRoleAdmin, &customer).Get(apitest.Applications), http.StatusOK)).
		To(Equal([]uuid.UUID{app}))
}

func TestLicensingAnswersUnknownReferencesWith400(t *testing.T) {
	knownBug(t, "an unknown application, version or customer violates a foreign key, which answers 500 "+
		"with the SQL error in the body")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	customer := vendor.CreateCustomerOrganization("Customer")
	g := NewWithT(t)

	for _, r := range []request{
		{
			"unknown application", http.MethodPost, apitest.ApplicationEntitlements,
			entitlementBody("a", uuid.New(), &customer),
		},
		{
			"unknown version", http.MethodPost, apitest.ApplicationEntitlements,
			entitlementBody("b", app, &customer, uuid.New()),
		},
		{"unknown customer", http.MethodPost, apitest.ApplicationEntitlements, entitlementBody("c", app, new(uuid.New()))},
		{
			"license key for unknown customer", http.MethodPost, apitest.LicenseKeys,
			licenseKeyBody("key", new(uuid.New()), object{}),
		},
	} {
		response := vendor.Do(r.method, r.path, r.body)
		g.Expect(response.StatusCode).To(Equal(http.StatusBadRequest), r.name)
		g.Expect(string(response.Body)).NotTo(Or(ContainSubstring(sqlErrorFragment), ContainSubstring(foreignKeyConstraint)),
			r.name)
	}
}

func TestApplicationEntitlementInUse(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	v1 := vendor.CreateApplicationVersion(app, "1.0.0")
	v2 := vendor.CreateApplicationVersion(app, "2.0.0")
	customer := vendor.CreateCustomerOrganization("Customer")
	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		entitlementBody("entitlement", app, &customer)), http.StatusOK).ID
	entitlementPath := apitest.Path(apitest.ApplicationEntitlements, entitlement)
	target := vendor.CreateDeploymentTarget("customer target", &customer)
	vendor.Deploy(api.DeploymentRequest{
		DeploymentTargetID: target, ApplicationVersionID: v2, ApplicationEntitlementID: &entitlement,
	})
	g := NewWithT(t)

	narrowed := vendor.Put(entitlementPath, entitlementBody("entitlement", app, &customer, v1))
	narrowed.ExpectStatus(http.StatusConflict)
	g.Expect(string(narrowed.Body)).To(ContainSubstring(`"customer target" uses version "2.0.0"`))
	vendor.Delete(entitlementPath).ExpectStatus(http.StatusBadRequest)
	g.Expect(apitest.QueryValue[int](h,
		`SELECT count(*) FROM ApplicationEntitlement_ApplicationVersion WHERE application_entitlement_id = $1`,
		entitlement)).To(BeZero())

	vendor.Put(entitlementPath, entitlementBody("entitlement", app, &customer, v2)).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[[]uuid.UUID](h,
		`SELECT array_agg(application_version_id) FROM ApplicationEntitlement_ApplicationVersion
		WHERE application_entitlement_id = $1`, entitlement)).To(Equal([]uuid.UUID{v2}))
}

func TestApplicationEntitlementAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		entitlementBody("entitlement", app, &customer)), http.StatusOK).ID
	otherEntitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		entitlementBody("other entitlement", app, &otherCustomer)), http.StatusOK).ID
	entitlementPath := apitest.Path(apitest.ApplicationEntitlements, entitlement)
	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	otherCustomerAdmin := vendor.Invite(types.UserRoleAdmin, &otherCustomer)
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	g := NewWithT(t)

	g.Expect(apitest.IDs(vendor.Get(apitest.ApplicationEntitlements), http.StatusOK)).
		To(ConsistOf(entitlement, otherEntitlement))
	g.Expect(apitest.IDs(customerAdmin.Get(apitest.ApplicationEntitlements), http.StatusOK)).
		To(Equal([]uuid.UUID{entitlement}))
	customerAdmin.Get(entitlementPath).ExpectStatus(http.StatusOK)
	reader.Get(entitlementPath).ExpectStatus(http.StatusOK)

	write := []request{
		{"create", http.MethodPost, apitest.ApplicationEntitlements, entitlementBody("new", app, &customer)},
		{"update", http.MethodPut, entitlementPath, entitlementBody("renamed", app, &customer)},
		{"delete", http.MethodDelete, entitlementPath, nil},
	}
	g.Expect(statuses(customerAdmin.Client, write)).To(HaveEach(http.StatusForbidden))
	g.Expect(statuses(reader.Client, write)).To(HaveEach(http.StatusForbidden))
	g.Expect(statuses(otherCustomerAdmin.Client, []request{
		{"get", http.MethodGet, entitlementPath, nil},
		{"delete", http.MethodDelete, entitlementPath, nil},
	})).To(HaveEach(http.StatusForbidden))
	g.Expect(statuses(h.RegisterVendor().Client, []request{
		{"get", http.MethodGet, entitlementPath, nil},
		{"update", http.MethodPut, entitlementPath, entitlementBody("renamed", app, &customer)},
		{"delete", http.MethodDelete, entitlementPath, nil},
	})).To(HaveEach(http.StatusForbidden))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM ApplicationEntitlement WHERE id = $1`, entitlement)).
		To(Equal("entitlement"))

	vendor.Delete(entitlementPath).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.IDs(customerAdmin.Get(apitest.ApplicationEntitlements), http.StatusOK)).To(BeEmpty())
}

func TestLicenseKeyErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	key := apitest.Expect[apitest.Entity](vendor.Post(apitest.LicenseKeys, licenseKeyBody("existing", &customer,
		object{"seats": 5})), http.StatusOK).ID
	keyPath := apitest.Path(apitest.LicenseKeys, key)
	unknownPath := apitest.Path(apitest.LicenseKeys, uuid.New())
	g := NewWithT(t)

	without := func(field string) object {
		body := licenseKeyBody("x", nil, object{})
		delete(body, field)
		return body
	}
	reversed := licenseKeyBody("x", nil, object{})
	reversed["notBefore"], reversed["expiresAt"] = reversed["expiresAt"], reversed["notBefore"]
	create := func(name string, body any) request { return request{name, http.MethodPost, apitest.LicenseKeys, body} }
	put := func(name string, body any) request { return request{name, http.MethodPut, keyPath, body} }
	g.Expect(statuses(vendor.Client, []request{
		create("no name", without("name")),
		create("no notBefore", without("notBefore")),
		create("no expiresAt", without("expiresAt")),
		create("expiresAt before notBefore", reversed),
		create("no payload", without("payload")),
		create("payload is no object", licenseKeyBody("x", nil, "seats")),
		create("payload with exp", licenseKeyBody("x", nil, object{"exp": 1})),
		create("payload with org", licenseKeyBody("x", nil, object{"org": "x"})),
		create("unknown template", object{"name": "x", "licenseTemplateId": uuid.New()}),
		create("duplicate name", licenseKeyBody("existing", &customer, object{})),
		create("malformed body", "{"),
		{"token of malformed id", http.MethodGet, apitest.LicenseKeys + "/nope/token", nil},
		{"token of unknown", http.MethodGet, unknownPath + "/token", nil},
		put("put expiresAt before notBefore", object{"expiresAt": time.Now().Add(-2 * time.Hour).UTC()}),
		put("put payload with exp", object{"payload": object{"exp": 1}}),
		put("put unknown template", object{"licenseTemplateId": uuid.New()}),
		put("put malformed body", "{"),
		{"put unknown", http.MethodPut, unknownPath, object{"description": "x"}},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"no name":                        http.StatusBadRequest,
		"no notBefore":                   http.StatusBadRequest,
		"no expiresAt":                   http.StatusBadRequest,
		"expiresAt before notBefore":     http.StatusBadRequest,
		"no payload":                     http.StatusBadRequest,
		"payload is no object":           http.StatusBadRequest,
		"payload with exp":               http.StatusBadRequest,
		"payload with org":               http.StatusBadRequest,
		"unknown template":               http.StatusBadRequest,
		"duplicate name":                 http.StatusBadRequest,
		"malformed body":                 http.StatusBadRequest,
		"token of malformed id":          http.StatusBadRequest,
		"token of unknown":               http.StatusNotFound,
		"put expiresAt before notBefore": http.StatusBadRequest,
		"put payload with exp":           http.StatusBadRequest,
		"put unknown template":           http.StatusBadRequest,
		"put malformed body":             http.StatusBadRequest,
		"put unknown":                    http.StatusNotFound,
		"delete unknown":                 http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseKey`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseKeyRevision WHERE license_key_id = $1`, key)).
		To(Equal(1))
}

func TestLicenseKeyTokens(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	key := apitest.Expect[apitest.Entity](vendor.Post(apitest.LicenseKeys, licenseKeyBody("key", &customer,
		object{"seats": 5})), http.StatusOK).ID
	vendorKey := apitest.Expect[apitest.Entity](vendor.Post(apitest.LicenseKeys, licenseKeyBody("vendor key", nil,
		object{})), http.StatusOK).ID
	keyPath := apitest.Path(apitest.LicenseKeys, key)
	g := NewWithT(t)

	publicKey, err := jwk.ParseKey(h.Anonymous().Get(licenseKeyPublicKey).ExpectStatus(http.StatusOK).Body,
		jwk.WithX509(true))
	g.Expect(err).NotTo(HaveOccurred())
	verify := func(client *apitest.Client) jwt.Token {
		t.Helper()
		response := apitest.Expect[struct {
			Token string `json:"token"`
		}](client.Get(keyPath+"/token"), http.StatusOK)
		token, err := jwt.Parse([]byte(response.Token), jwt.WithKey(jwa.EdDSA(), publicKey))
		g.Expect(err).NotTo(HaveOccurred())
		return token
	}
	claim := func(token jwt.Token, name string) any {
		t.Helper()
		value, err := jwt.Get[any](token, name)
		g.Expect(err).NotTo(HaveOccurred())
		return value
	}

	token := verify(vendor.Client)
	subject, _ := token.Subject()
	g.Expect(subject).To(Equal(key.String()))
	g.Expect(claim(token, "org")).To(Equal(vendor.OrganizationID.String()))
	g.Expect(claim(token, "seats")).To(BeEquivalentTo(5))

	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	g.Expect(claim(verify(customerAdmin.Client), "seats")).To(BeEquivalentTo(5))
	g.Expect(apitest.IDs(customerAdmin.Get(apitest.LicenseKeys), http.StatusOK)).To(Equal([]uuid.UUID{key}))
	g.Expect(statuses(customerAdmin.Client, []request{
		{"token of a vendor key", http.MethodGet, apitest.Path(apitest.LicenseKeys, vendorKey) + "/token", nil},
	})).To(HaveEach(http.StatusNotFound))
	customerAdmin.Put(keyPath, object{"description": "x"}).ExpectStatus(http.StatusForbidden)
	g.Expect(statuses(vendor.Invite(types.UserRoleAdmin, &otherCustomer).Client, []request{
		{"token", http.MethodGet, keyPath + "/token", nil},
		{"revisions", http.MethodGet, keyPath + "/revisions", nil},
	})).To(HaveEach(http.StatusNotFound))
	g.Expect(statuses(h.RegisterVendor().Client, []request{
		{"token", http.MethodGet, keyPath + "/token", nil},
		{"update", http.MethodPut, keyPath, object{"description": "x"}},
		{"delete", http.MethodDelete, keyPath, nil},
	})).To(HaveEach(http.StatusNotFound))

	vendor.Put(keyPath, object{"description": "only metadata"}).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM LicenseKeyRevision WHERE license_key_id = $1`, key)).
		To(Equal(1))
	vendor.Put(keyPath, object{"payload": object{"seats": 10}}).ExpectStatus(http.StatusOK)
	g.Expect(claim(verify(vendor.Client), "seats")).To(BeEquivalentTo(10))
	revisions := apitest.Expect[[]struct {
		Token string `json:"token"`
	}](vendor.Get(keyPath+"/revisions"), http.StatusOK)
	g.Expect(revisions).To(HaveLen(2))
	for _, revision := range revisions {
		_, err := jwt.Parse([]byte(revision.Token), jwt.WithKey(jwa.EdDSA(), publicKey))
		g.Expect(err).NotTo(HaveOccurred())
	}

	vendor.Delete(keyPath).ExpectStatus(http.StatusNoContent)
	vendor.Get(keyPath + "/token").ExpectStatus(http.StatusNotFound)
}
