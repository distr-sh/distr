package customdomains

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func domain(domainType types.DomainType, name string) types.CustomDomain {
	return types.CustomDomain{Domain: name, Type: domainType}
}

func TestCustomerPortalDomainOrDefault(t *testing.T) {
	t.Run("the customer's own portal domain wins over the vendor's", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(customerPortalDomainOrDefault(
			[]types.CustomDomain{domain(types.DomainTypeCustomerPortal, "acme.customer.com")},
			[]types.CustomDomain{
				domain(types.DomainTypeApp, "app.vendor.com"),
				domain(types.DomainTypeCustomerPortal, "portal.vendor.com"),
			},
			nil,
		)).To(Equal("https://acme.customer.com"))
	})

	t.Run("falls back to the vendor's shared portal domain", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(customerPortalDomainOrDefault(
			nil,
			[]types.CustomDomain{
				domain(types.DomainTypeApp, "app.vendor.com"),
				domain(types.DomainTypeCustomerPortal, "portal.vendor.com"),
			},
			nil,
		)).To(Equal("https://portal.vendor.com"))
	})

	t.Run("falls back to the vendor's app domain", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(customerPortalDomainOrDefault(
			nil,
			[]types.CustomDomain{domain(types.DomainTypeApp, "app.vendor.com")},
			nil,
		)).To(Equal("https://app.vendor.com"))
	})

	// Skipping the legacy branding step would move every vendor that has not migrated to a
	// self-service domain off their own hostname in customer mail.
	t.Run("falls back to the legacy branding app domain before the instance default", func(t *testing.T) {
		g := NewWithT(t)
		legacy := "legacy.vendor.com"
		g.Expect(customerPortalDomainOrDefault(
			nil, nil, &types.OrganizationBranding{AppDomain: &legacy},
		)).To(Equal("https://legacy.vendor.com"))
	})
}

// A customer_portal row must never be picked up by the vendor-facing resolvers, or a customer
// hostname would end up in agent manifests and in the registry host.
func TestVendorResolversIgnoreCustomerPortalDomains(t *testing.T) {
	portalOnly := []types.CustomDomain{domain(types.DomainTypeCustomerPortal, "portal.vendor.com")}

	t.Run("app domain", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(appDomainOrDefault(portalOnly, nil)).NotTo(ContainSubstring("portal.vendor.com"))
	})

	t.Run("registry domain", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(registryDomainOrDefault(portalOnly, nil)).NotTo(ContainSubstring("portal.vendor.com"))
	})
}
