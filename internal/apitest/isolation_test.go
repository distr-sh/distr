package apitest_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

type vendorResources struct {
	application, deploymentTarget, secret, customerOrganization, member uuid.UUID
}

func createVendorResources(vendor *apitest.User) vendorResources {
	application := vendor.CreateApplication("original")
	vendor.CreateApplicationVersion(application, "1.0.0")
	return vendorResources{
		application:          application,
		deploymentTarget:     vendor.CreateDeploymentTarget("original", nil),
		secret:               vendor.CreateSecret("ORIGINAL", "original", nil),
		customerOrganization: vendor.CreateCustomerOrganization("original"),
		member:               vendor.Invite(types.UserRoleReadWrite, nil).ID,
	}
}

type request struct {
	name, method, path string
	body               any
}

// statuses sends every request as client and maps its name to the status of the response, so
// that one assertion reports every deviation at once.
func statuses(client *apitest.Client, requests []request) map[string]int {
	result := make(map[string]int, len(requests))
	for _, r := range requests {
		result[r.name] = client.Do(r.method, r.path, r.body).StatusCode
	}
	return result
}

func allWithStatus(requests []request, status int) map[string]int {
	result := make(map[string]int, len(requests))
	for _, r := range requests {
		result[r.name] = status
	}
	return result
}

// mutations are valid requests changing or deleting every resource in r, so that a refusal cannot
// come from validating the body.
func (r vendorResources) mutations() []request {
	application := apitest.Path(apitest.Applications, r.application)
	deploymentTarget := apitest.Path(apitest.DeploymentTargets, r.deploymentTarget)
	secret := apitest.Path(apitest.Secrets, r.secret)
	customerOrganization := apitest.Path(apitest.CustomerOrganizations, r.customerOrganization)
	return []request{
		{"update application", http.MethodPut, application, api.UpdateApplicationRequest{Name: "changed"}},
		{"patch application", http.MethodPatch, application, api.PatchApplicationRequest{Name: new("changed")}},
		{
			"create application version", http.MethodPost, application + "/versions",
			apitest.DockerApplicationVersion("2.0.0"),
		},
		{"delete application", http.MethodDelete, application, nil},
		{
			"update deployment target", http.MethodPut, deploymentTarget,
			api.UpdateDeploymentTargetRequest{DeploymentTargetSettings: api.DeploymentTargetSettings{Name: "changed"}},
		},
		{"request deployment target access", http.MethodPost, deploymentTarget + "/access-request", nil},
		{"delete deployment target", http.MethodDelete, deploymentTarget, nil},
		{"update secret", http.MethodPut, secret, api.UpdateSecretRequest{Value: "changed"}},
		{"delete secret", http.MethodDelete, secret, nil},
		{"delete customer organization", http.MethodDelete, customerOrganization, nil},
		{"delete user account", http.MethodDelete, apitest.Path(apitest.UserAccounts, r.member), nil},
	}
}

// expectUnchanged fails unless every resource in r still exists as created.
func (r vendorResources) expectUnchanged(t *testing.T, h *apitest.Harness) {
	t.Helper()
	g := NewWithT(t)
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM Application WHERE id = $1`, r.application)).
		To(Equal("original"))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ApplicationVersion WHERE application_id = $1`,
		r.application)).To(Equal(1))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM DeploymentTarget WHERE id = $1`, r.deploymentTarget)).
		To(Equal("original"))
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT access_key_hash IS NULL AND pending_access_key_hash IS NULL FROM DeploymentTarget WHERE id = $1`,
		r.deploymentTarget,
	)).To(BeTrue(), "an access request issues a new key")
	g.Expect(apitest.QueryValue[string](h, `SELECT key FROM Secret WHERE id = $1`, r.secret)).To(Equal("ORIGINAL"))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM CustomerOrganization WHERE id = $1`,
		r.customerOrganization)).To(Equal("original"))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM UserAccount WHERE id = $1`, r.member)).To(Equal(1))
}

func TestOtherVendorCannotSeeOrChangeResources(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	owner := h.RegisterVendor()
	resources := createVendorResources(owner)
	intruder := h.RegisterVendor()

	requests := slices.Concat([]request{
		{"get application", http.MethodGet, apitest.Path(apitest.Applications, resources.application), nil},
		{
			"get deployment target", http.MethodGet,
			apitest.Path(apitest.DeploymentTargets, resources.deploymentTarget), nil,
		},
	}, resources.mutations())
	g := NewWithT(t)
	g.Expect(statuses(intruder.Client, requests)).To(Equal(allWithStatus(requests, http.StatusNotFound)))
	resources.expectUnchanged(t, h)

	for _, list := range []string{
		apitest.Applications,
		apitest.DeploymentTargets,
		apitest.Secrets,
		apitest.CustomerOrganizations,
	} {
		g.Expect(apitest.IDs(intruder.Get(list), http.StatusOK)).To(BeEmpty(), list)
	}
	g.Expect(apitest.IDs(intruder.Get(apitest.UserAccounts), http.StatusOK)).To(Equal([]uuid.UUID{intruder.ID}))
}

func TestOtherVendorCannotUpdateCustomerOrganization(t *testing.T) {
	knownBug(t, "answers 500 instead of 404 for an unknown or another vendor's customer organization "+
		"because the handler does not map apierrors.ErrNotFound")
	t.Parallel()
	h := apitest.New(t)
	owner := h.RegisterVendor()
	customerOrganization := owner.CreateCustomerOrganization("original")

	h.RegisterVendor().Put(apitest.Path(apitest.CustomerOrganizations, customerOrganization),
		api.CreateUpdateCustomerOrganizationRequest{Name: "changed"}).ExpectStatus(http.StatusNotFound)
	owner.Put(apitest.Path(apitest.CustomerOrganizations, uuid.New()),
		api.CreateUpdateCustomerOrganizationRequest{Name: "changed"}).ExpectStatus(http.StatusNotFound)
	NewWithT(t).Expect(apitest.QueryValue[string](h, `SELECT name FROM CustomerOrganization WHERE id = $1`,
		customerOrganization)).To(Equal("original"))
}

func TestReadOnlyUserCannotChangeResources(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	resources := createVendorResources(vendor)
	reader := vendor.Invite(types.UserRoleReadOnly, nil)

	requests := slices.Concat([]request{
		{
			"create application", http.MethodPost, apitest.Applications,
			api.CreateApplicationRequest{Name: "new", Type: types.DeploymentTypeDocker},
		},
		{"create secret", http.MethodPost, apitest.Secrets, api.CreateSecretRequest{Key: "NEW", Value: "new"}},
		{
			"create customer organization", http.MethodPost, apitest.CustomerOrganizations,
			api.CreateUpdateCustomerOrganizationRequest{Name: "new"},
		},
		{"invite user", http.MethodPost, apitest.UserAccounts, api.CreateUserAccountRequest{
			Email: "new@distr.test", Name: "New", UserRole: types.UserRoleAdmin,
		}},
		{
			"update customer organization", http.MethodPut,
			apitest.Path(apitest.CustomerOrganizations, resources.customerOrganization),
			api.CreateUpdateCustomerOrganizationRequest{Name: "changed"},
		},
	}, resources.mutations())
	g := NewWithT(t)
	g.Expect(statuses(reader.Client, requests)).To(Equal(allWithStatus(requests, http.StatusForbidden)))
	resources.expectUnchanged(t, h)

	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Application`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Secret`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM CustomerOrganization`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM UserAccount WHERE email = 'new@distr.test'`)).
		To(Equal(0))

	g.Expect(apitest.IDs(reader.Get(apitest.Applications), http.StatusOK)).
		To(Equal([]uuid.UUID{resources.application}))
	g.Expect(apitest.IDs(reader.Get(apitest.Secrets), http.StatusOK)).To(Equal([]uuid.UUID{resources.secret}))
}
