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

func TestCustomerSeesOnlyItsOwnResources(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	own := vendor.CreateCustomerOrganization("own")
	other := vendor.CreateCustomerOrganization("other")
	ownTarget := vendor.CreateDeploymentTarget("own", &own)
	otherTarget := vendor.CreateDeploymentTarget("other", &other)
	vendorTarget := vendor.CreateDeploymentTarget("vendor", nil)
	ownSecret := vendor.CreateSecret("OWN", "own", &own)
	otherSecret := vendor.CreateSecret("OTHER", "other", &other)
	vendor.CreateSecret("VENDOR", "vendor", nil)
	customer := vendor.Invite(types.UserRoleAdmin, &own)

	g := NewWithT(t)
	g.Expect(apitest.IDs(customer.Get(apitest.DeploymentTargets), http.StatusOK)).To(Equal([]uuid.UUID{ownTarget}))
	g.Expect(apitest.IDs(customer.Get(apitest.Secrets), http.StatusOK)).To(Equal([]uuid.UUID{ownSecret}))
	g.Expect(apitest.IDs(customer.Get(apitest.UserAccounts), http.StatusOK)).To(Equal([]uuid.UUID{customer.ID}))

	g.Expect(statuses(customer.Client, []request{
		{
			"get other customer's deployment target", http.MethodGet,
			apitest.Path(apitest.DeploymentTargets, otherTarget), nil,
		},
		{"get vendor's deployment target", http.MethodGet, apitest.Path(apitest.DeploymentTargets, vendorTarget), nil},
		{
			"delete other customer's deployment target", http.MethodDelete,
			apitest.Path(apitest.DeploymentTargets, otherTarget), nil,
		},
		{
			"update other customer's secret", http.MethodPut, apitest.Path(apitest.Secrets, otherSecret),
			api.UpdateSecretRequest{Value: "changed"},
		},
		{"delete other customer's secret", http.MethodDelete, apitest.Path(apitest.Secrets, otherSecret), nil},
		{
			"create secret for other customer", http.MethodPost, apitest.Secrets,
			api.CreateSecretRequest{Key: "SMUGGLED", Value: "value", CustomerOrganizationID: &other},
		},
		{"list customer organizations", http.MethodGet, apitest.CustomerOrganizations, nil},
		{
			"create application", http.MethodPost, apitest.Applications,
			api.CreateApplicationRequest{Name: "new", Type: types.DeploymentTypeDocker},
		},
	})).To(Equal(map[string]int{
		"get other customer's deployment target":    http.StatusNotFound,
		"get vendor's deployment target":            http.StatusNotFound,
		"delete other customer's deployment target": http.StatusNotFound,
		"update other customer's secret":            http.StatusNotFound,
		"delete other customer's secret":            http.StatusNotFound,
		"create secret for other customer":          http.StatusBadRequest,
		"list customer organizations":               http.StatusForbidden,
		"create application":                        http.StatusForbidden,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentTarget WHERE id = $1`, otherTarget)).
		To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Secret WHERE id = $1`, otherSecret)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Secret WHERE key = 'SMUGGLED'`)).To(Equal(0))

	target := customer.CreateDeploymentTarget("created by customer", nil)
	secret := customer.CreateSecret("CREATED_BY_CUSTOMER", "value", nil)
	g.Expect(apitest.QueryValue[uuid.UUID](h, `SELECT customer_organization_id FROM DeploymentTarget WHERE id = $1`,
		target)).To(Equal(own))
	g.Expect(apitest.QueryValue[uuid.UUID](h, `SELECT customer_organization_id FROM Secret WHERE id = $1`, secret)).
		To(Equal(own))
}

func TestEntitlementsLimitTheApplicationsACustomerSees(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	entitled := vendor.CreateApplication("entitled")
	hidden := vendor.CreateApplication("hidden")
	own := vendor.CreateCustomerOrganization("own")
	other := vendor.CreateCustomerOrganization("other")
	customer := vendor.Invite(types.UserRoleAdmin, &own)
	otherCustomer := vendor.Invite(types.UserRoleAdmin, &other)

	g := NewWithT(t)
	g.Expect(apitest.IDs(customer.Get(apitest.Applications), http.StatusOK)).
		To(ConsistOf(entitled, hidden), "without any entitlement a customer sees every application")

	apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		types.ApplicationEntitlementWithVersions{ApplicationEntitlementBase: types.ApplicationEntitlementBase{
			Name:                   "entitlement",
			ApplicationID:          entitled,
			CustomerOrganizationID: &own,
		}},
	), http.StatusOK)

	g.Expect(apitest.IDs(customer.Get(apitest.Applications), http.StatusOK)).To(Equal([]uuid.UUID{entitled}))
	customer.Get(apitest.Path(apitest.Applications, entitled)).ExpectStatus(http.StatusOK)
	g.Expect(apitest.IDs(otherCustomer.Get(apitest.Applications), http.StatusOK)).To(BeEmpty())
	g.Expect(apitest.IDs(vendor.Get(apitest.Applications), http.StatusOK)).To(ConsistOf(entitled, hidden))
}

func TestCustomerCannotGetApplicationItIsNotEntitledTo(t *testing.T) {
	knownBug(t, "answers 500 instead of 404 because db.GetApplicationWithEntitlementOwnerID "+
		"indexes the empty result of pgx.CollectRows")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	entitled := vendor.CreateApplication("entitled")
	hidden := vendor.CreateApplication("hidden")
	own := vendor.CreateCustomerOrganization("own")
	other := vendor.CreateCustomerOrganization("other")
	customer := vendor.Invite(types.UserRoleAdmin, &own)
	otherCustomer := vendor.Invite(types.UserRoleAdmin, &other)
	apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		types.ApplicationEntitlementWithVersions{ApplicationEntitlementBase: types.ApplicationEntitlementBase{
			Name:                   "entitlement",
			ApplicationID:          entitled,
			CustomerOrganizationID: &own,
		}},
	), http.StatusOK)

	customer.Get(apitest.Path(apitest.Applications, hidden)).ExpectStatus(http.StatusNotFound)
	otherCustomer.Get(apitest.Path(apitest.Applications, entitled)).ExpectStatus(http.StatusNotFound)
}
