package handlers

import (
	"testing"

	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/env"
	. "github.com/onsi/gomega"
)

type organizationCreatorAuth struct {
	authinfo.AuthInfo
	superAdmin         bool
	organizationScoped bool
}

func (a organizationCreatorAuth) IsSuperAdmin() bool       { return a.superAdmin }
func (a organizationCreatorAuth) OrganizationScoped() bool { return a.organizationScoped }

func TestMayCreateOrganization(t *testing.T) {
	vendor := organizationCreatorAuth{}
	superAdmin := organizationCreatorAuth{superAdmin: true}

	t.Run("vendors create organizations only while registration is enabled", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(mayCreateOrganization(vendor, true, env.RegistrationEnabled)).To(BeTrue())
		g.Expect(mayCreateOrganization(vendor, true, env.RegistrationHidden)).To(BeFalse())
		g.Expect(mayCreateOrganization(vendor, true, env.RegistrationDisabled)).To(BeFalse())
		g.Expect(mayCreateOrganization(vendor, false, env.RegistrationEnabled)).To(BeFalse())
	})

	t.Run("super admins create organizations whatever the registration mode", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(mayCreateOrganization(superAdmin, false, env.RegistrationDisabled)).To(BeTrue())
	})

	t.Run("a credential confined to one organization creates none", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(mayCreateOrganization(organizationCreatorAuth{organizationScoped: true}, true, env.RegistrationEnabled)).
			To(BeFalse())
		g.Expect(mayCreateOrganization(
			organizationCreatorAuth{superAdmin: true, organizationScoped: true}, false, env.RegistrationEnabled,
		)).To(BeFalse())
	})
}
