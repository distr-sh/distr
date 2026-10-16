package apitest_test

import (
	"net/http"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

// registryVendor is a vendor whose organization has a slug and who holds an access token, which is
// what pushing to the registry takes.
type registryVendor struct {
	*apitest.User
	h    *apitest.Harness
	slug string
	key  string
}

func newRegistryVendor(h *apitest.Harness, slug string) registryVendor {
	vendor := h.RegisterVendor()
	vendor.SetSlug(slug)
	return registryVendor{User: vendor, h: h, slug: slug, key: vendor.CreateAccessToken()}
}

func (v registryVendor) repository(artifact string) string {
	return v.slug + "/" + artifact
}

// push pushes an image of one layer holding layer as artifact:tag and returns the ID of the artifact.
func (v registryVendor) push(t *testing.T, artifact, tag, layer string) uuid.UUID {
	t.Helper()
	_, err := v.h.PushImage(v.key, v.repository(artifact), tag, layer)
	NewWithT(t).Expect(err).NotTo(HaveOccurred(), "push of %v:%v", artifact, tag)
	return apitest.QueryValue[uuid.UUID](v.h,
		`SELECT id FROM Artifact WHERE organization_id = $1 AND name = $2`, v.OrganizationID, artifact)
}

type artifactDetail struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Public   bool      `json:"public"`
	Versions []struct {
		ID     uuid.UUID `json:"id"`
		Digest string    `json:"digest"`
		Tags   []struct {
			Name string `json:"name"`
		} `json:"tags"`
	} `json:"versions"`
}

// tags maps every tag of the artifact to the ID of the version it points at, which is the ID an
// artifact entitlement selects.
func (a artifactDetail) tags() map[string]uuid.UUID {
	result := map[string]uuid.UUID{}
	for _, version := range a.Versions {
		for _, tag := range version.Tags {
			result[tag.Name] = version.ID
		}
	}
	return result
}

// digest is the digest of the manifest tag points at, or an empty string if there is no such tag.
func (a artifactDetail) digest(tag string) string {
	for _, version := range a.Versions {
		if version.ID == a.tags()[tag] {
			return version.Digest
		}
	}
	return ""
}

func getArtifact(client *apitest.Client, id uuid.UUID) artifactDetail {
	return apitest.Expect[artifactDetail](client.Get(apitest.Path(apitest.Artifacts, id)), http.StatusOK)
}

func tagPath(artifact uuid.UUID, tag string) string {
	return apitest.Path(apitest.Artifacts, artifact) + "/tags/" + tag
}

func TestArtifactLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	g := NewWithT(t)

	withoutSlug := vendor.Post(apitest.Artifacts, api.CreateArtifactRequest{Name: "app"})
	withoutSlug.ExpectStatus(http.StatusBadRequest)
	g.Expect(string(withoutSlug.Body)).To(ContainSubstring("slug"))

	vendor.SetSlug("vendor")
	created := apitest.Expect[apitest.Entity](vendor.Post(apitest.Artifacts, api.CreateArtifactRequest{Name: "app"}),
		http.StatusCreated).ID
	vendor.Post(apitest.Artifacts, api.CreateArtifactRequest{Name: "app"}).ExpectStatus(http.StatusConflict)
	g.Expect(apitest.IDs(vendor.Get(apitest.Artifacts), http.StatusOK)).To(Equal([]uuid.UUID{created}))
	g.Expect(getArtifact(vendor.Client, created).Versions).To(BeEmpty())

	key := vendor.CreateAccessToken()
	_, err := h.PushImage(key, "vendor/app", "1.0.0", "layer")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(apitest.IDs(vendor.Get(apitest.Artifacts), http.StatusOK)).To(Equal([]uuid.UUID{created}),
		"a push to the name of an existing artifact adds a version to it")
	g.Expect(getArtifact(vendor.Client, created).tags()).To(HaveKey("1.0.0"))

	vendor.Delete(apitest.Path(apitest.Artifacts, created)).ExpectStatus(http.StatusNoContent)
	vendor.Get(apitest.Path(apitest.Artifacts, created)).ExpectStatus(http.StatusNotFound)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Artifact`)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ArtifactVersion`)).To(BeZero())
	g.Expect(h.PullImage(key, "vendor/app", "1.0.0")).NotTo(Succeed())
}

func TestArtifactErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "layer")
	artifactPath := apitest.Path(apitest.Artifacts, artifact)
	unknownPath := apitest.Path(apitest.Artifacts, uuid.New())
	g := NewWithT(t)

	create := func(name string, body any) request { return request{name, http.MethodPost, apitest.Artifacts, body} }
	patch := func(name string, body any) request { return request{name, http.MethodPatch, artifactPath, body} }
	g.Expect(statuses(vendor.Client, []request{
		create("create without name", api.CreateArtifactRequest{}),
		create("create with an empty upstream URL", api.CreateArtifactRequest{Name: "cache", UpstreamURL: new("")}),
		create("create with upstream credentials but no upstream URL", api.CreateArtifactRequest{
			Name:         "cache",
			UpstreamAuth: &api.ArtifactUpstreamAuth{Type: types.UpstreamAuthTypeBasic},
		}),
		create("create malformed body", "{"),
		{"get malformed id", http.MethodGet, apitest.Artifacts + "/nope", nil},
		{"get unknown", http.MethodGet, unknownPath, nil},
		patch("patch the upstream of an artifact that is no pull-through cache",
			object{"upstreamUrl": "ghcr.io/distr-sh/distr"}),
		patch("patch an empty upstream URL", object{"upstreamUrl": ""}),
		patch("patch malformed body", "{"),
		{"patch unknown", http.MethodPatch, unknownPath, object{"public": true}},
		{"sync an artifact that is no pull-through cache", http.MethodPost, artifactPath + "/sync", nil},
		{"delete the last tag", http.MethodDelete, tagPath(artifact, "1.0.0"), nil},
		{"delete an unknown tag", http.MethodDelete, tagPath(artifact, "2.0.0"), nil},
		{"delete a tag of an unknown artifact", http.MethodDelete, unknownPath + "/tags/1.0.0", nil},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"create without name":                                  http.StatusBadRequest,
		"create with an empty upstream URL":                    http.StatusBadRequest,
		"create with upstream credentials but no upstream URL": http.StatusBadRequest,
		"create malformed body":                                http.StatusBadRequest,
		"get malformed id":                                     http.StatusNotFound,
		"get unknown":                                          http.StatusNotFound,
		"patch the upstream of an artifact that is no pull-through cache": http.StatusBadRequest,
		"patch an empty upstream URL":                                     http.StatusBadRequest,
		"patch malformed body":                                            http.StatusBadRequest,
		"patch unknown":                                                   http.StatusNotFound,
		"sync an artifact that is no pull-through cache":                  http.StatusBadRequest,
		"delete the last tag":                                             http.StatusConflict,
		"delete an unknown tag":                                           http.StatusNotFound,
		"delete a tag of an unknown artifact":                             http.StatusNotFound,
		"delete unknown":                                                  http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Artifact`)).To(Equal(1))
	g.Expect(getArtifact(vendor.Client, artifact).tags()).To(HaveKey("1.0.0"))
	g.Expect(getArtifact(vendor.Client, artifact).Public).To(BeFalse())
}

func TestArtifactAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "layer")
	vendor.push(t, "app", "latest", "layer")
	artifactPath := apitest.Path(apitest.Artifacts, artifact)
	customer := vendor.CreateCustomerOrganization("Customer")
	g := NewWithT(t)

	writes := []request{
		{"create", http.MethodPost, apitest.Artifacts, api.CreateArtifactRequest{Name: "new"}},
		{"make public", http.MethodPatch, artifactPath, object{"public": true}},
		{"delete a tag", http.MethodDelete, tagPath(artifact, "latest"), nil},
		{"delete", http.MethodDelete, artifactPath, nil},
	}
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	g.Expect(apitest.IDs(reader.Get(apitest.Artifacts), http.StatusOK)).To(Equal([]uuid.UUID{artifact}))
	reader.Get(artifactPath).ExpectStatus(http.StatusOK)
	g.Expect(statuses(reader.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	customerAdmin.Get(artifactPath).ExpectStatus(http.StatusOK)
	g.Expect(statuses(customerAdmin.Client, writes)).To(Equal(allWithStatus(writes, http.StatusForbidden)))
	g.Expect(statuses(vendor.Invite(types.UserRoleReadWrite, nil).Client, writes[1:2])).
		To(Equal(allWithStatus(writes[1:2], http.StatusForbidden)), "only an admin may make an artifact public")

	other := h.RegisterVendor()
	otherCustomer := other.CreateCustomerOrganization("Other Customer")
	foreign := []request{
		{"get", http.MethodGet, artifactPath, nil},
		{"make public", http.MethodPatch, artifactPath, object{"public": true}},
		{"sync", http.MethodPost, artifactPath + "/sync", nil},
		{"delete a tag", http.MethodDelete, tagPath(artifact, "latest"), nil},
		{"delete", http.MethodDelete, artifactPath, nil},
	}
	g.Expect(statuses(other.Client, foreign)).To(Equal(allWithStatus(foreign, http.StatusNotFound)))
	g.Expect(apitest.IDs(other.Get(apitest.Artifacts), http.StatusOK)).To(BeEmpty())
	other.Invite(types.UserRoleAdmin, &otherCustomer).Get(artifactPath).ExpectStatus(http.StatusNotFound)

	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Artifact`)).To(Equal(1))
	g.Expect(getArtifact(vendor.Client, artifact).tags()).To(HaveKey("latest"))
	g.Expect(getArtifact(vendor.Client, artifact).Public).To(BeFalse())

	vendor.Patch(artifactPath, object{"public": true}).ExpectStatus(http.StatusOK)
	g.Expect(getArtifact(vendor.Client, artifact).Public).To(BeTrue())
	vendor.Delete(tagPath(artifact, "latest")).ExpectStatus(http.StatusNoContent)
	g.Expect(getArtifact(vendor.Client, artifact).tags()).NotTo(HaveKey("latest"))
}

func TestArtifactInUseByAnEntitlement(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "first")
	vendor.push(t, "app", "stable", "first")
	vendor.push(t, "app", "2.0.0", "second")
	artifactPath := apitest.Path(apitest.Artifacts, artifact)
	customer := vendor.CreateCustomerOrganization("Customer")
	versions := getArtifact(vendor.Client, artifact).tags()
	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ArtifactEntitlements,
		artifactEntitlementBody("entitlement", &customer, artifactSelection(artifact, versions["1.0.0"]))),
		http.StatusOK).ID
	g := NewWithT(t)

	vendor.Delete(tagPath(artifact, "1.0.0")).ExpectStatus(http.StatusNoContent)
	lastTag := vendor.Delete(tagPath(artifact, "stable"))
	lastTag.ExpectStatus(http.StatusBadRequest)
	g.Expect(string(lastTag.Body)).To(ContainSubstring("entitlement"))
	vendor.Delete(tagPath(artifact, "2.0.0")).ExpectStatus(http.StatusNoContent)
	inUse := vendor.Delete(artifactPath)
	inUse.ExpectStatus(http.StatusBadRequest)
	g.Expect(string(inUse.Body)).To(ContainSubstring("entitlement"))
	g.Expect(getArtifact(vendor.Client, artifact).tags()).To(Equal(map[string]uuid.UUID{"stable": versions["1.0.0"]}))

	vendor.Delete(apitest.Path(apitest.ArtifactEntitlements, entitlement)).ExpectStatus(http.StatusNoContent)
	vendor.Delete(artifactPath).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Artifact`)).To(BeZero())
}
