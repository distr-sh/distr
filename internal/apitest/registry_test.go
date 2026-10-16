package apitest_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type tagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type catalog struct {
	Repositories []string `json:"repositories"`
}

func manifestPath(repository, reference string) string {
	return "/v2/" + repository + "/manifests/" + reference
}

func blobPath(repository string, d digest.Digest) string {
	return "/v2/" + repository + "/blobs/" + d.String()
}

func tagsPath(repository string) string {
	return "/v2/" + repository + "/tags/list"
}

func TestRegistryPushAndPull(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	g := NewWithT(t)

	manifest, err := h.PushImage(vendor.key, "vendor/app", "1.0.0", "layer")
	g.Expect(err).NotTo(HaveOccurred())
	artifact := apitest.QueryValue[uuid.UUID](h, `SELECT id FROM Artifact WHERE name = 'app'`)
	detail := getArtifact(vendor.Client, artifact)
	g.Expect(detail.Name).To(Equal("app"))
	g.Expect(detail.Versions).To(HaveLen(1))
	g.Expect(detail.Versions[0].Digest).To(Equal(manifest.String()))
	g.Expect(detail.tags()).To(Equal(map[string]uuid.UUID{"1.0.0": detail.Versions[0].ID}))
	g.Expect(apitest.QueryValue[[]string](h,
		`SELECT array_agg(name ORDER BY name) FROM ArtifactVersion WHERE created_by_useraccount_id = $1`, vendor.ID,
	)).To(Equal([]string{"1.0.0", manifest.String()}), "a push stores the tag and the digest it points at")

	g.Expect(h.PullImage(vendor.key, "vendor/app", "1.0.0")).To(Succeed())
	g.Expect(h.PullImage(vendor.key, "vendor/app", manifest.String())).To(Succeed())
	registry := h.Registry(vendor.key)
	g.Expect(apitest.Expect[tagList](registry.Get(tagsPath("vendor/app")), http.StatusOK)).
		To(Equal(tagList{Name: "vendor/app", Tags: []string{"1.0.0"}}))
	g.Expect(apitest.Expect[catalog](registry.Get("/v2/_catalog"), http.StatusOK).Repositories).
		To(Equal([]string{"vendor/app"}))
	registry.Get(manifestPath("vendor/app", "2.0.0")).ExpectStatus(http.StatusNotFound)
	registry.Get(manifestPath("vendor/unknown", "1.0.0")).ExpectStatus(http.StatusNotFound)
	registry.Get(blobPath("vendor/app", digest.FromString("unknown"))).ExpectStatus(http.StatusNotFound)

	g.Expect(apitest.QueryValue[bool](h,
		`SELECT count(*) > 0 AND bool_and(useraccount_id = $1 AND NOT anonymous AND customer_organization_id IS NULL)
		FROM ArtifactVersionPull`, vendor.ID)).To(BeTrue(), "every pull is recorded for the user who pulled")
	pulls := apitest.Expect[[]api.ArtifactVersionPullResponse](vendor.Get(apitest.ArtifactPulls), http.StatusOK)
	g.Expect(pulls).NotTo(BeEmpty())
	g.Expect(pulls[0].UserAccountEmail).To(HaveValue(Equal(vendor.Email)))
	g.Expect(pulls[0].Artifact.Name).To(Equal("app"))
}

func TestRegistryAuthentication(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	vendor.push(t, "app", "1.0.0", "layer")
	deleted := apitest.Expect[api.AccessTokenWithKey](vendor.Post(apitest.AccessTokens,
		api.CreateAccessTokenRequest{Label: new("deleted")}), http.StatusCreated)
	vendor.Delete(apitest.Path(apitest.AccessTokens, deleted.ID)).ExpectStatus(http.StatusNoContent)
	g := NewWithT(t)

	requests := []request{
		{"version check", http.MethodGet, "/v2/", nil},
		{"manifest", http.MethodGet, manifestPath("vendor/app", "1.0.0"), nil},
		{"catalog", http.MethodGet, "/v2/_catalog", nil},
	}
	for name, client := range map[string]*apitest.Client{
		"no credentials":                              h.Registry(""),
		"a deleted access token":                      h.Registry(deleted.Key),
		"a malformed password":                        h.Registry("nope"),
		"the login token of the user as password":     h.Registry(vendor.Token),
		"the login token of the user as Bearer token": h.Registry("").WithHeader("Authorization", "Bearer "+vendor.Token),
	} {
		g.Expect(statuses(client, requests)).To(Equal(allWithStatus(requests, http.StatusUnauthorized)), name)
	}
	g.Expect(h.Registry("").Get("/v2/").Header.Get("WWW-Authenticate")).To(HavePrefix("Basic"),
		"an OCI client sends its credentials only after a challenge")
	g.Expect(statuses(h.Registry(vendor.key), requests)).To(Equal(allWithStatus(requests, http.StatusOK)))
}

func TestRegistryWriteAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "layer")
	customer := vendor.CreateCustomerOrganization("Customer")
	cappedKey := apitest.Expect[api.AccessTokenWithKey](vendor.Post(apitest.AccessTokens,
		api.CreateAccessTokenRequest{UserRole: new(types.UserRoleReadOnly)}), http.StatusCreated).Key
	other := newRegistryVendor(h, "other")
	controllerToken := h.ControllerToken(vendor.RequestAccess(vendor.CreateDeploymentTarget("own", nil)))
	g := NewWithT(t)

	for name, key := range map[string]string{
		"a read-only user":                  vendor.Invite(types.UserRoleReadOnly, nil).CreateAccessToken(),
		"a token capped to read-only":       cappedKey,
		"a customer admin":                  vendor.Invite(types.UserRoleAdmin, &customer).CreateAccessToken(),
		"the controller of a vendor target": controllerToken,
		"another vendor":                    other.key,
	} {
		_, err := h.PushImage(key, "vendor/app", "2.0.0", "pushed by "+name)
		g.Expect(apitest.RegistryStatus(err)).To(Equal(http.StatusForbidden), name)
		g.Expect(h.Registry(key).Delete(manifestPath("vendor/app", "1.0.0")).StatusCode).
			To(Equal(http.StatusForbidden), name)
	}
	_, err := h.PushImage("", "vendor/app", "2.0.0", "pushed anonymously")
	g.Expect(apitest.RegistryStatus(err)).To(Equal(http.StatusUnauthorized))
	g.Expect(getArtifact(vendor.Client, artifact).tags()).To(HaveLen(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Artifact`)).To(Equal(1))

	_, err = h.PushImage(vendor.Invite(types.UserRoleReadWrite, nil).CreateAccessToken(), "vendor/app", "2.0.0", "new")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(getArtifact(vendor.Client, artifact).tags()).To(HaveKey("2.0.0"))
}

func TestRegistryOrganizationIsolation(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	vendor.push(t, "app", "1.0.0", "layer of the vendor")
	other := newRegistryVendor(h, "other")
	other.push(t, "app", "1.0.0", "layer of the other vendor")
	layer := digest.FromString("layer of the vendor")
	g := NewWithT(t)

	g.Expect(statuses(h.Registry(other.key), []request{
		{"manifest", http.MethodGet, manifestPath("vendor/app", "1.0.0"), nil},
		{"tags", http.MethodGet, tagsPath("vendor/app"), nil},
		{"blob through the name of the artifact", http.MethodGet, blobPath("vendor/app", layer), nil},
		{"blob through an own artifact", http.MethodGet, blobPath("other/app", layer), nil},
		{"delete", http.MethodDelete, manifestPath("vendor/app", "1.0.0"), nil},
	})).To(Equal(map[string]int{
		"manifest":                              http.StatusForbidden,
		"tags":                                  http.StatusForbidden,
		"blob through the name of the artifact": http.StatusForbidden,
		"blob through an own artifact":          http.StatusNotFound,
		"delete":                                http.StatusForbidden,
	}))
	g.Expect(apitest.RegistryStatus(h.PullImage(other.key, "vendor/app", "1.0.0"))).To(Equal(http.StatusForbidden))
	g.Expect(apitest.Expect[catalog](h.Registry(other.key).Get("/v2/_catalog"), http.StatusOK).Repositories).
		To(Equal([]string{"other/app"}))
	g.Expect(h.PullImage(vendor.key, "vendor/app", "1.0.0")).To(Succeed())
}

func TestRegistryBlobsAreOnlyReachableThroughTheirUploader(t *testing.T) {
	knownBug(t, "a manifest push does not check that the blobs it references were uploaded by or are "+
		"visible to the pushing organization, so whoever knows the digest of another organization's "+
		"private layer can push a manifest referencing it and then download the layer")
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	vendor.push(t, "app", "1.0.0", "secret layer")
	other := newRegistryVendor(h, "other")
	layer := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Digest:    digest.FromString("secret layer"),
		Size:      int64(len("secret layer")),
	}
	manifest, err := json.Marshal(ocispec.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    ocispec.DescriptorEmptyJSON,
		Layers:    []ocispec.Descriptor{layer},
	})
	g := NewWithT(t)
	g.Expect(err).NotTo(HaveOccurred())

	h.Registry(other.key).WithHeader("Content-Type", ocispec.MediaTypeImageManifest).
		Do(http.MethodPut, manifestPath("other/stolen", "1.0.0"), json.RawMessage(manifest))
	g.Expect(h.Registry(other.key).Get(blobPath("other/stolen", layer.Digest)).Body).
		NotTo(Equal([]byte("secret layer")))
}

func TestRegistryPublicArtifact(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	artifact := vendor.push(t, "app", "1.0.0", "public layer")
	vendor.push(t, "private", "1.0.0", "private layer")
	g := NewWithT(t)

	g.Expect(apitest.RegistryStatus(h.PullImage("", "vendor/app", "1.0.0"))).To(Equal(http.StatusUnauthorized))
	vendor.Patch(apitest.Path(apitest.Artifacts, artifact), object{"public": true}).
		ExpectStatus(http.StatusOK)

	g.Expect(h.PullImage("", "vendor/app", "1.0.0")).To(Succeed())
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT count(*) > 0 AND bool_and(anonymous AND useraccount_id IS NULL) FROM ArtifactVersionPull`)).
		To(BeTrue(), "an anonymous pull is recorded as such")
	anonymous := h.Registry("")
	g.Expect(statuses(anonymous, []request{
		{"tags of the public artifact", http.MethodGet, tagsPath("vendor/app"), nil},
		{"manifest of a private artifact", http.MethodGet, manifestPath("vendor/private", "1.0.0"), nil},
		{"tags of a private artifact", http.MethodGet, tagsPath("vendor/private"), nil},
		{
			"layer of a private artifact through the public artifact", http.MethodGet,
			blobPath("vendor/app", digest.FromString("private layer")), nil,
		},
		{"catalog", http.MethodGet, "/v2/_catalog", nil},
		{"upload to the public artifact", http.MethodPost, "/v2/vendor/app/blobs/uploads/", nil},
		{"delete a tag of the public artifact", http.MethodDelete, manifestPath("vendor/app", "1.0.0"), nil},
	})).To(Equal(map[string]int{
		"tags of the public artifact":                             http.StatusOK,
		"manifest of a private artifact":                          http.StatusUnauthorized,
		"tags of a private artifact":                              http.StatusUnauthorized,
		"layer of a private artifact through the public artifact": http.StatusUnauthorized,
		"catalog":                             http.StatusUnauthorized,
		"upload to the public artifact":       http.StatusUnauthorized,
		"delete a tag of the public artifact": http.StatusUnauthorized,
	}))

	customer := vendor.CreateCustomerOrganization("Customer")
	vendor.Post(apitest.ArtifactEntitlements, artifactEntitlementBody("private only", &customer,
		artifactSelection(apitest.QueryValue[uuid.UUID](h, `SELECT id FROM Artifact WHERE name = 'private'`)))).
		ExpectStatus(http.StatusOK)
	otherCustomer := vendor.Invite(types.UserRoleAdmin, new(vendor.CreateCustomerOrganization("Other Customer")))
	g.Expect(h.PullImage(otherCustomer.CreateAccessToken(), "vendor/app", "1.0.0")).To(Succeed(),
		"a public artifact needs no entitlement")
	g.Expect(apitest.IDs(otherCustomer.Get(apitest.Artifacts), http.StatusOK)).To(Equal([]uuid.UUID{artifact}))

	vendor.Patch(apitest.Path(apitest.Artifacts, artifact), object{"public": false}).
		ExpectStatus(http.StatusOK)
	g.Expect(apitest.RegistryStatus(h.PullImage("", "vendor/app", "1.0.0"))).To(Equal(http.StatusUnauthorized))
}

func TestRegistryCustomerEntitlements(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	app := vendor.push(t, "app", "1.0.0", "first")
	vendor.push(t, "app", "2.0.0", "second")
	tool := vendor.push(t, "tool", "1.0.0", "tool")
	customer := vendor.CreateCustomerOrganization("Customer")
	customerUser := vendor.Invite(types.UserRoleReadOnly, &customer)
	customerKey := customerUser.CreateAccessToken()
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	otherCustomerUser := vendor.Invite(types.UserRoleReadOnly, &otherCustomer)
	otherKey := otherCustomerUser.CreateAccessToken()
	detail := getArtifact(vendor.Client, app)
	versions := detail.tags()
	g := NewWithT(t)

	g.Expect(h.PullImage(otherKey, "vendor/tool", "1.0.0")).To(Succeed(),
		"until a vendor creates its first artifact entitlement, every customer may pull every artifact")
	g.Expect(apitest.IDs(otherCustomerUser.Get(apitest.Artifacts), http.StatusOK)).To(ConsistOf(app, tool))

	entitlement := apitest.Expect[apitest.Entity](vendor.Post(apitest.ArtifactEntitlements,
		artifactEntitlementBody("first only", &customer, artifactSelection(app, versions["1.0.0"]))),
		http.StatusOK).ID
	g.Expect(h.PullImage(customerKey, "vendor/app", "1.0.0")).To(Succeed())
	g.Expect(h.PullImage(customerKey, "vendor/app", detail.digest("1.0.0"))).To(Succeed())
	customerRegistry := h.Registry(customerKey)
	g.Expect(statuses(customerRegistry, []request{
		{"another version", http.MethodGet, manifestPath("vendor/app", "2.0.0"), nil},
		{"another artifact", http.MethodGet, manifestPath("vendor/tool", "1.0.0"), nil},
		{"layer of another version", http.MethodGet, blobPath("vendor/app", digest.FromString("second")), nil},
		{"layer of another artifact", http.MethodGet, blobPath("vendor/tool", digest.FromString("tool")), nil},
	})).To(HaveEach(http.StatusForbidden))
	g.Expect(apitest.Expect[tagList](customerRegistry.Get(tagsPath("vendor/app")), http.StatusOK).Tags).
		To(Equal([]string{"1.0.0"}))
	g.Expect(apitest.IDs(customerUser.Get(apitest.Artifacts), http.StatusOK)).To(Equal([]uuid.UUID{app}))
	g.Expect(getArtifact(customerUser.Client, app).tags()).To(Equal(map[string]uuid.UUID{"1.0.0": versions["1.0.0"]}))
	customerUser.Get(apitest.Path(apitest.Artifacts, tool)).ExpectStatus(http.StatusNotFound)

	g.Expect(apitest.RegistryStatus(h.PullImage(otherKey, "vendor/tool", "1.0.0"))).To(Equal(http.StatusForbidden))
	g.Expect(apitest.IDs(otherCustomerUser.Get(apitest.Artifacts), http.StatusOK)).To(BeEmpty())
	otherCustomerUser.Get(apitest.Path(apitest.Artifacts, app)).ExpectStatus(http.StatusNotFound)

	entitlementPath := apitest.Path(apitest.ArtifactEntitlements, entitlement)
	vendor.Put(entitlementPath, artifactEntitlementBody("whole app", &customer, artifactSelection(app))).
		ExpectStatus(http.StatusOK)
	g.Expect(h.PullImage(customerKey, "vendor/app", "2.0.0")).To(Succeed())
	g.Expect(apitest.RegistryStatus(h.PullImage(customerKey, "vendor/tool", "1.0.0"))).To(Equal(http.StatusForbidden))

	expired := artifactEntitlementBody("whole app", &customer, artifactSelection(app))
	expired["expiresAt"] = time.Now().Add(-time.Hour).UTC()
	vendor.Put(entitlementPath, expired).ExpectStatus(http.StatusOK)
	g.Expect(apitest.RegistryStatus(h.PullImage(customerKey, "vendor/app", "1.0.0"))).To(Equal(http.StatusForbidden))
	g.Expect(apitest.IDs(customerUser.Get(apitest.Artifacts), http.StatusOK)).To(BeEmpty())

	g.Expect(apitest.QueryValue[[]uuid.UUID](h,
		`SELECT array_agg(DISTINCT customer_organization_id) FROM ArtifactVersionPull WHERE useraccount_id = $1`,
		customerUser.ID)).To(Equal([]uuid.UUID{customer}), "a customer's pull is recorded for the customer")
}

func TestRegistryHonorsTheArtifactsFeatureOfACustomer(t *testing.T) {
	knownBug(t, "the artifacts feature of a customer organization only hides the sidebar entry, the API and "+
		"the registry serve the customer's users all the same")
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	vendor.push(t, "app", "1.0.0", "layer")
	customer := apitest.Expect[apitest.Entity](vendor.Post(apitest.CustomerOrganizations,
		api.CreateUpdateCustomerOrganizationRequest{
			Name:     "Customer",
			Features: []types.CustomerOrganizationFeature{types.CustomerOrganizationFeatureDeploymentTargets},
		}), http.StatusOK).ID
	customerUser := vendor.Invite(types.UserRoleAdmin, &customer)
	g := NewWithT(t)

	g.Expect(customerUser.Get(apitest.Artifacts).StatusCode).To(Equal(http.StatusForbidden))
	g.Expect(apitest.RegistryStatus(h.PullImage(customerUser.CreateAccessToken(), "vendor/app", "1.0.0"))).
		To(Equal(http.StatusForbidden))
}

func TestRegistryControllerPulls(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	app := vendor.push(t, "app", "1.0.0", "layer")
	vendor.push(t, "tool", "1.0.0", "tool")
	customer := vendor.CreateCustomerOrganization("Customer")
	target := vendor.CreateDeploymentTarget("customer target", &customer)
	token := h.ControllerToken(vendor.RequestAccess(target))
	vendor.Post(apitest.ArtifactEntitlements, artifactEntitlementBody("app", &customer, artifactSelection(app))).
		ExpectStatus(http.StatusOK)
	g := NewWithT(t)

	g.Expect(h.PullImage(token, "vendor/app", "1.0.0")).To(Succeed())
	g.Expect(apitest.RegistryStatus(h.PullImage(token, "vendor/tool", "1.0.0"))).To(Equal(http.StatusForbidden),
		"a controller of a customer is held to the customer's entitlements")
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT count(*) > 0 AND bool_and(deployment_target_id = $1 AND customer_organization_id = $2)
		FROM ArtifactVersionPull`, target, customer)).To(BeTrue())

	pulls := apitest.Expect[[]api.ArtifactVersionPullResponse](vendor.Get(apitest.ArtifactPulls), http.StatusOK)
	g.Expect(pulls).NotTo(BeEmpty())
	g.Expect(pulls[0].DeploymentTargetName).To(HaveValue(Equal("customer target")))
	g.Expect(pulls[0].CustomerOrganizationName).To(HaveValue(Equal("Customer")))

	other := newRegistryVendor(h, "other")
	other.push(t, "app", "1.0.0", "other layer")
	g.Expect(apitest.RegistryStatus(h.PullImage(token, "other/app", "1.0.0"))).To(Equal(http.StatusForbidden))
}

func TestRegistryTags(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := newRegistryVendor(h, "vendor")
	first, err := h.PushImage(vendor.key, "vendor/app", "1.0.0", "first")
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	artifact := apitest.QueryValue[uuid.UUID](h, `SELECT id FROM Artifact`)
	g := NewWithT(t)

	_, err = h.PushImage(vendor.key, "vendor/app", "1.0.0", "changed")
	g.Expect(apitest.RegistryStatus(err)).To(Equal(http.StatusConflict), "a tag is immutable")
	again, err := h.PushImage(vendor.key, "vendor/app", "1.0.0", "first")
	g.Expect(err).NotTo(HaveOccurred(), "pushing a tag again with the same content is no change")
	g.Expect(again).To(Equal(first))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ArtifactVersion`)).To(Equal(2))

	registry := h.Registry(vendor.key)
	g.Expect(statuses(registry, []request{
		{"delete by digest", http.MethodDelete, manifestPath("vendor/app", first.String()), nil},
		{"delete the last tag", http.MethodDelete, manifestPath("vendor/app", "1.0.0"), nil},
		{"delete an unknown tag", http.MethodDelete, manifestPath("vendor/app", "2.0.0"), nil},
		{"delete a tag of an unknown name", http.MethodDelete, manifestPath("vendor/unknown", "1.0.0"), nil},
	})).To(Equal(map[string]int{
		"delete by digest":                http.StatusBadRequest,
		"delete the last tag":             http.StatusConflict,
		"delete an unknown tag":           http.StatusNotFound,
		"delete a tag of an unknown name": http.StatusNotFound,
	}))

	_, err = h.PushImage(vendor.key, "vendor/app", "latest", "first")
	g.Expect(err).NotTo(HaveOccurred())
	registry.Delete(manifestPath("vendor/app", "1.0.0")).ExpectStatus(http.StatusAccepted)
	g.Expect(getArtifact(vendor.Client, artifact).tags()).To(HaveKey("latest"))
	g.Expect(getArtifact(vendor.Client, artifact).tags()).NotTo(HaveKey("1.0.0"))
	g.Expect(h.PullImage(vendor.key, "vendor/app", first.String())).To(Succeed(),
		"the digest stays while a tag points at it")
}
