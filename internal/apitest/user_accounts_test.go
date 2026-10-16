package apitest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func createUser(name, email string, role types.UserRole, customerOrganizationID *uuid.UUID) request {
	return request{name, http.MethodPost, apitest.UserAccounts, api.CreateUserAccountRequest{
		Email: email, Name: name, UserRole: role, CustomerOrganizationID: customerOrganizationID,
	}}
}

func patchUser(name string, id uuid.UUID, body any) request {
	return request{name, http.MethodPatch, apitest.Path(apitest.UserAccounts, id), body}
}

func deleteUser(name string, id uuid.UUID) request {
	return request{name, http.MethodDelete, apitest.Path(apitest.UserAccounts, id), nil}
}

func TestUserAccountValidation(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	otherCustomer := h.RegisterVendor().CreateCustomerOrganization("Other Customer")
	member := vendor.Invite(types.UserRoleReadOnly, nil)
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		createUser("duplicate email", member.Email, types.UserRoleReadWrite, nil),
		createUser("invalid email", "nobody", types.UserRoleReadWrite, nil),
		createUser("no role", "norole@distr.test", "", nil),
		createUser("unknown role", "unknownrole@distr.test", "superadmin", nil),
		createUser("other vendor's customer", "foreign@distr.test", types.UserRoleAdmin, &otherCustomer),
		createUser("unknown customer", "unknown@distr.test", types.UserRoleAdmin, new(uuid.New())),
		{"create malformed body", http.MethodPost, apitest.UserAccounts, "{"},
		patchUser("unknown role on patch", member.ID, api.PatchUserAccountRequest{UserRole: new(types.UserRole("x"))}),
		patchUser("patch malformed body", member.ID, "{"),
		{"patch malformed id", http.MethodPatch, apitest.UserAccounts + "/nope", api.PatchUserAccountRequest{}},
		patchUser("patch unknown user", uuid.New(), api.PatchUserAccountRequest{Name: new("x")}),
		deleteUser("delete unknown user", uuid.New()),
		patchUser("patch own role", vendor.ID, api.PatchUserAccountRequest{UserRole: new(types.UserRoleReadOnly)}),
		deleteUser("delete self", vendor.ID),
	})).To(Equal(map[string]int{
		"duplicate email":         http.StatusBadRequest,
		"invalid email":           http.StatusBadRequest,
		"no role":                 http.StatusBadRequest,
		"unknown role":            http.StatusBadRequest,
		"other vendor's customer": http.StatusBadRequest,
		"unknown customer":        http.StatusBadRequest,
		"create malformed body":   http.StatusBadRequest,
		"unknown role on patch":   http.StatusBadRequest,
		"patch malformed body":    http.StatusBadRequest,
		"patch malformed id":      http.StatusNotFound,
		"patch unknown user":      http.StatusNotFound,
		"delete unknown user":     http.StatusNotFound,
		"patch own role":          http.StatusForbidden,
		"delete self":             http.StatusForbidden,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Organization_UserAccount WHERE organization_id = $1`,
		vendor.OrganizationID)).To(Equal(2))
	g.Expect(apitest.QueryValue[types.UserRole](h,
		`SELECT user_role FROM Organization_UserAccount WHERE organization_id = $1 AND user_account_id = $2`,
		vendor.OrganizationID, member.ID)).To(Equal(types.UserRoleReadOnly))

	renamed := apitest.Expect[api.UserAccountResponse](vendor.Patch(apitest.Path(apitest.UserAccounts, vendor.ID),
		api.PatchUserAccountRequest{Name: new("Renamed")}), http.StatusOK)
	g.Expect(renamed.Name).To(Equal("Renamed"))
}

func TestRoleChangeAppliesToTheNextLogin(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	promoted := vendor.Invite(types.UserRoleReadOnly, nil)
	demoted := vendor.Invite(types.UserRoleReadWrite, nil)
	g := NewWithT(t)

	setRole := func(user *apitest.User, role types.UserRole) {
		t.Helper()
		patched := apitest.Expect[api.UserAccountResponse](vendor.Patch(apitest.Path(apitest.UserAccounts, user.ID),
			api.PatchUserAccountRequest{UserRole: &role}), http.StatusOK)
		g.Expect(patched.UserRole).To(Equal(role))
	}
	createApplication := func(client *apitest.Client) *apitest.Response {
		return client.Post(apitest.Applications,
			api.CreateApplicationRequest{Name: "app", Type: types.DeploymentTypeDocker})
	}
	relogin := func(user *apitest.User) *apitest.Client {
		t.Helper()
		return h.WithBearer(apitest.Expect[api.AuthLoginResponse](
			login(h, user.Email, apitest.Password, nil), http.StatusOK).Token)
	}

	setRole(promoted, types.UserRoleReadWrite)
	createApplication(promoted.Client).ExpectStatus(http.StatusForbidden)
	createApplication(relogin(promoted)).ExpectStatus(http.StatusOK)

	setRole(demoted, types.UserRoleReadOnly)
	demoted.Get(apitest.Applications).ExpectStatus(http.StatusUnauthorized)
	createApplication(relogin(demoted)).ExpectStatus(http.StatusForbidden)
}

func TestPatchUserAccountAnswersAMalformedBodyOnce(t *testing.T) {
	knownBug(t, "patchUserAccountHandler writes a second error after JsonBody already answered, "+
		"so the body holds the message twice")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	member := vendor.Invite(types.UserRoleReadOnly, nil)

	response := vendor.Patch(apitest.Path(apitest.UserAccounts, member.ID), "{").ExpectStatus(http.StatusBadRequest)
	NewWithT(t).Expect(strings.Count(string(response.Body), "\n")).To(Equal(1))
}

func TestUserAccountPermissions(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	vendorReadWrite := vendor.Invite(types.UserRoleReadWrite, nil)
	vendorReadOnly := vendor.Invite(types.UserRoleReadOnly, nil)
	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	customerReadWrite := vendor.Invite(types.UserRoleReadWrite, &customer)
	customerReadOnly := vendor.Invite(types.UserRoleReadOnly, &customer)
	otherCustomerUser := vendor.Invite(types.UserRoleReadOnly, &otherCustomer)
	g := NewWithT(t)

	rename := api.PatchUserAccountRequest{Name: new("Renamed")}
	g.Expect(statuses(vendorReadWrite.Client, []request{
		createUser("create vendor user", "rw-vendor@distr.test", types.UserRoleReadOnly, nil),
		patchUser("patch vendor user", vendorReadOnly.ID, rename),
		deleteUser("delete vendor user", vendorReadOnly.ID),
		createUser("create customer user", "rw-customer@distr.test", types.UserRoleReadOnly, &customer),
		patchUser("patch customer user", customerReadOnly.ID,
			api.PatchUserAccountRequest{UserRole: new(types.UserRoleReadWrite)}),
		deleteUser("delete customer user", otherCustomerUser.ID),
	})).To(Equal(map[string]int{
		"create vendor user":   http.StatusForbidden,
		"patch vendor user":    http.StatusForbidden,
		"delete vendor user":   http.StatusForbidden,
		"create customer user": http.StatusOK,
		"patch customer user":  http.StatusOK,
		"delete customer user": http.StatusNoContent,
	}))
	g.Expect(statuses(vendorReadOnly.Client, []request{
		createUser("create customer user", "ro-customer@distr.test", types.UserRoleReadOnly, &customer),
		patchUser("patch customer user", customerReadOnly.ID, rename),
		deleteUser("delete customer user", customerReadOnly.ID),
	})).To(HaveEach(http.StatusForbidden))

	for _, user := range []*apitest.User{customerReadWrite, customerReadOnly} {
		g.Expect(statuses(user.Client, []request{
			createUser("create user", "customer-"+uuid.NewString()+"@distr.test", types.UserRoleReadOnly, nil),
			patchUser("patch user", customerAdmin.ID, rename),
			deleteUser("delete user", customerAdmin.ID),
		})).To(HaveEach(http.StatusForbidden), user.Email)
	}

	g.Expect(statuses(customerAdmin.Client, []request{
		patchUser("patch vendor user", vendorReadWrite.ID, rename),
		deleteUser("delete vendor user", vendorReadWrite.ID),
		patchUser("patch own role", customerAdmin.ID, api.PatchUserAccountRequest{UserRole: new(types.UserRoleReadOnly)}),
		patchUser("patch customer user", customerReadWrite.ID,
			api.PatchUserAccountRequest{UserRole: new(types.UserRoleReadOnly)}),
		deleteUser("delete customer user", customerReadOnly.ID),
	})).To(Equal(map[string]int{
		"patch vendor user":    http.StatusNotFound,
		"delete vendor user":   http.StatusNotFound,
		"patch own role":       http.StatusForbidden,
		"patch customer user":  http.StatusOK,
		"delete customer user": http.StatusNoContent,
	}))

	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Organization_UserAccount WHERE user_account_id = ANY($1)`,
		[]uuid.UUID{vendorReadOnly.ID, customerAdmin.ID, vendorReadWrite.ID})).To(Equal(3))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Organization_UserAccount WHERE user_account_id = ANY($1)`,
		[]uuid.UUID{customerReadOnly.ID, otherCustomerUser.ID})).To(BeZero())
}

func TestCustomerAdminCreatesUsersInItsOwnCustomerOrganization(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	otherCustomer := vendor.CreateCustomerOrganization("Other Customer")
	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	g := NewWithT(t)

	created := apitest.Expect[api.CreateUserAccountResponse](customerAdmin.Post(apitest.UserAccounts,
		api.CreateUserAccountRequest{
			Email:                  "customer-user-" + uuid.NewString() + "@distr.test",
			UserRole:               types.UserRoleReadOnly,
			CustomerOrganizationID: &otherCustomer,
		}), http.StatusOK)
	g.Expect(apitest.QueryValue[uuid.UUID](h,
		`SELECT customer_organization_id FROM Organization_UserAccount WHERE user_account_id = $1`,
		created.User.ID)).To(Equal(customer))
	g.Expect(apitest.IDs(customerAdmin.Get(apitest.UserAccounts), http.StatusOK)).
		To(ConsistOf(customerAdmin.ID, created.User.ID))
}

func TestDeletedUserLosesAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	member := vendor.Invite(types.UserRoleReadWrite, nil)
	access := apitest.Expect[api.AccessTokenWithKey](
		member.Post(apitest.AccessTokens, api.CreateAccessTokenRequest{}), http.StatusCreated)
	token := h.WithAccessToken(access.Key)
	token.Get(apitest.Applications).ExpectStatus(http.StatusOK)
	g := NewWithT(t)

	vendor.Delete(apitest.Path(apitest.UserAccounts, member.ID)).ExpectStatus(http.StatusNoContent)
	vendor.Delete(apitest.Path(apitest.UserAccounts, member.ID)).ExpectStatus(http.StatusNotFound)

	g.Expect(statuses(member.Client, []request{
		{"context", http.MethodGet, contextPath, nil},
		{"applications", http.MethodGet, apitest.Applications, nil},
	})).To(HaveEach(http.StatusUnauthorized))
	token.Get(apitest.Applications).ExpectStatus(http.StatusUnauthorized)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AccessToken WHERE user_account_id = $1`, member.ID)).
		To(BeZero())
	g.Expect(apitest.IDs(vendor.Get(apitest.UserAccounts), http.StatusOK)).To(Equal([]uuid.UUID{vendor.ID}))
}
