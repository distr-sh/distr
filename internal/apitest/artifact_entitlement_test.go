package apitest_test

import (
	"net/http"
	"testing"

	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

// artifactSelection selects an artifact and, if versions are given, only those versions of it, which
// is how an artifact entitlement body names what it covers.
func artifactSelection(artifact uuid.UUID, versions ...uuid.UUID) object {
	selection := object{"artifactId": artifact}
	if len(versions) > 0 {
		selection["versionIds"] = versions
	}
	return selection
}

func artifactEntitlementBody(name string, customer *uuid.UUID, selections ...object) object {
	body := object{"name": name}
	if customer != nil {
		body["customerOrganizationId"] = *customer
	}
	if len(selections) > 0 {
		body["artifacts"] = selections
	}
	return body
}

func TestArtifactEntitlementLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	app := vendor.push(t, "app", "1.0.0", "first")
	vendor.push(t, "app", "2.0.0", "second")
	tool := vendor.push(t, "tool", "1.0.0", "tool")
	versions := getArtifact(vendor.Client, app).tags()
	customer := vendor.CreateCustomerOrganization("Customer")
	g := NewWithT(t)
	selections := func(id uuid.UUID) []string {
		return apitest.QueryValue[[]string](h,
			`SELECT array_agg(artifact_id::text || ':' || coalesce(artifact_version_id::text, 'all') ORDER BY 1)
			FROM ArtifactEntitlement_Artifact WHERE artifact_entitlement_id = $1`, id)
	}

	created := apitest.Expect[struct {
		ID        uuid.UUID `json:"id"`
		Artifacts []struct {
			ArtifactID uuid.UUID   `json:"artifactId"`
			VersionIDs []uuid.UUID `json:"versionIds"`
		} `json:"artifacts"`
	}](vendor.Post(apitest.ArtifactEntitlements, artifactEntitlementBody("entitlement", &customer,
		artifactSelection(app, versions["1.0.0"]))), http.StatusOK)
	g.Expect(created.Artifacts).To(HaveLen(1))
	g.Expect(created.Artifacts[0].ArtifactID).To(Equal(app))
	g.Expect(created.Artifacts[0].VersionIDs).To(Equal([]uuid.UUID{versions["1.0.0"]}))
	g.Expect(apitest.IDs(vendor.Get(apitest.ArtifactEntitlements), http.StatusOK)).To(Equal([]uuid.UUID{created.ID}))
	path := apitest.Path(apitest.ArtifactEntitlements, created.ID)

	vendor.Put(path, artifactEntitlementBody("renamed", &customer,
		artifactSelection(app, versions["1.0.0"], versions["2.0.0"]), artifactSelection(tool))).
		ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM ArtifactEntitlement WHERE id = $1`, created.ID)).
		To(Equal("renamed"))
	g.Expect(selections(created.ID)).To(ConsistOf(
		app.String()+":"+versions["1.0.0"].String(),
		app.String()+":"+versions["2.0.0"].String(),
		tool.String()+":all",
	))

	vendor.Put(path, artifactEntitlementBody("renamed", &customer, artifactSelection(tool))).
		ExpectStatus(http.StatusOK)
	g.Expect(selections(created.ID)).To(Equal([]string{tool.String() + ":all"}))

	vendor.Delete(path).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ArtifactEntitlement_Artifact`)).To(BeZero())
	g.Expect(apitest.IDs(vendor.Get(apitest.ArtifactEntitlements), http.StatusOK)).To(BeEmpty())
}

func TestArtifactEntitlementErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "layer")
	customer := vendor.CreateCustomerOrganization("Customer")
	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ArtifactEntitlements,
		artifactEntitlementBody("existing", &customer, artifactSelection(artifact))), http.StatusOK).ID
	entitlementPath := apitest.Path(apitest.ArtifactEntitlements, entitlement)
	unknownPath := apitest.Path(apitest.ArtifactEntitlements, uuid.New())
	g := NewWithT(t)

	create := func(name string, body any) request {
		return request{name, http.MethodPost, apitest.ArtifactEntitlements, body}
	}
	put := func(name string, body any) request { return request{name, http.MethodPut, entitlementPath, body} }
	g.Expect(statuses(vendor.Client, []request{
		create("no name", artifactEntitlementBody("", &customer, artifactSelection(artifact))),
		create("the same artifact twice", artifactEntitlementBody("dup artifact", &customer,
			artifactSelection(artifact), artifactSelection(artifact))),
		create("duplicate name for the same customer",
			artifactEntitlementBody("existing", &customer, artifactSelection(artifact))),
		create("malformed body", "{"),
		{"put malformed id", http.MethodPut, apitest.ArtifactEntitlements + "/nope", artifactEntitlementBody("x", &customer)},
		put("put without name", artifactEntitlementBody("", &customer, artifactSelection(artifact))),
		put("put to another id", object{"id": uuid.New(), "name": "existing"}),
		put("put malformed body", "{"),
		{"put unknown", http.MethodPut, unknownPath, artifactEntitlementBody("x", &customer)},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"no name":                              http.StatusBadRequest,
		"the same artifact twice":              http.StatusBadRequest,
		"duplicate name for the same customer": http.StatusBadRequest,
		"malformed body":                       http.StatusBadRequest,
		"put malformed id":                     http.StatusBadRequest,
		"put without name":                     http.StatusBadRequest,
		"put to another id":                    http.StatusBadRequest,
		"put malformed body":                   http.StatusBadRequest,
		"put unknown":                          http.StatusNotFound,
		"delete unknown":                       http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ArtifactEntitlement`)).To(Equal(1))

	vendor.Post(apitest.ArtifactEntitlements,
		artifactEntitlementBody("existing", new(vendor.CreateCustomerOrganization("Other")),
			artifactSelection(artifact))).ExpectStatus(http.StatusOK)
}

func TestArtifactEntitlementAnswersUnknownReferencesWith400(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "layer")
	customer := vendor.CreateCustomerOrganization("Customer")
	g := NewWithT(t)

	for _, r := range []request{
		{
			"unknown artifact", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("a", &customer, artifactSelection(uuid.New())),
		},
		{
			"unknown version", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("b", &customer, artifactSelection(artifact, uuid.New())),
		},
		{
			"unknown customer", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("c", new(uuid.New()), artifactSelection(artifact)),
		},
	} {
		response := vendor.Do(r.method, r.path, r.body)
		g.Expect(response.StatusCode).To(Equal(http.StatusBadRequest), r.name)
		g.Expect(string(response.Body)).NotTo(Or(ContainSubstring(sqlErrorFragment), ContainSubstring("_fkey")), r.name)
	}
}

func TestArtifactEntitlementReferencesOnlyResourcesOfTheOwnOrganization(t *testing.T) {
	knownBug(t, "creating an artifact entitlement does not check that the artifact, versions and "+
		"customer belong to the organization, so a vendor can reference another vendor's and then read "+
		"their names through the entitlement")
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	customer := vendor.CreateCustomerOrganization("Customer")
	other := newRegistryVendor(h, "other")
	foreignArtifact := other.push(t, "app", "1.0.0", "layer")
	foreignVersion := getArtifact(other.Client, foreignArtifact).tags()["1.0.0"]
	foreignCustomer := other.CreateCustomerOrganization("Foreign Customer")
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		{
			"another vendor's artifact", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("a", &customer, artifactSelection(foreignArtifact)),
		},
		{
			"another vendor's version", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("b", &customer, artifactSelection(foreignArtifact, foreignVersion)),
		},
		{
			"another vendor's customer", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("c", &foreignCustomer),
		},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ArtifactEntitlement`)).To(BeZero())
}

func TestArtifactEntitlementAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "layer")
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ArtifactEntitlements,
		artifactEntitlementBody("entitlement", &customer, artifactSelection(artifact))), http.StatusOK).ID
	otherEntitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ArtifactEntitlements,
		artifactEntitlementBody("other", &otherCustomer, artifactSelection(artifact))), http.StatusOK).ID
	entitlementPath := apitest.Path(apitest.ArtifactEntitlements, entitlement)
	g := NewWithT(t)

	g.Expect(apitest.IDs(vendor.Get(apitest.ArtifactEntitlements), http.StatusOK)).
		To(ConsistOf(entitlement, otherEntitlement))

	write := []request{
		{
			"create", http.MethodPost, apitest.ArtifactEntitlements,
			artifactEntitlementBody("new", &customer, artifactSelection(artifact)),
		},
		{"update", http.MethodPut, entitlementPath, artifactEntitlementBody("renamed", &customer,
			artifactSelection(artifact))},
		{"delete", http.MethodDelete, entitlementPath, nil},
	}
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	g.Expect(statuses(reader.Client, write)).To(Equal(allWithStatus(write, http.StatusForbidden)))
	customerRequests := append([]request{{"list", http.MethodGet, apitest.ArtifactEntitlements, nil}}, write...)
	g.Expect(statuses(vendor.Invite(types.UserRoleAdmin, &customer).Client, customerRequests)).
		To(Equal(allWithStatus(customerRequests, http.StatusForbidden)))

	other := h.RegisterVendor()
	foreign := []request{
		{"update", http.MethodPut, entitlementPath, artifactEntitlementBody("renamed", &customer,
			artifactSelection(artifact))},
		{"delete", http.MethodDelete, entitlementPath, nil},
	}
	g.Expect(statuses(other.Client, foreign)).To(Equal(allWithStatus(foreign, http.StatusNotFound)))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM ArtifactEntitlement WHERE id = $1`, entitlement)).
		To(Equal("entitlement"))

	vendor.Delete(entitlementPath).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.IDs(vendor.Get(apitest.ArtifactEntitlements), http.StatusOK)).To(Equal([]uuid.UUID{otherEntitlement}))
}
