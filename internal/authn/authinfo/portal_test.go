package authinfo

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestMayEnterCustomerPortal(t *testing.T) {
	partnerID, otherPartnerID, customerID := uuid.New(), uuid.New(), uuid.New()
	partner := Portal{PartnerOrganizationID: &partnerID}
	customer := Portal{CustomerOrganizationID: &customerID}

	t.Run("vendor members and super admins enter every customer", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnterCustomerPortal(Portal{}, false, &partnerID)).To(BeTrue())
		g.Expect(MayEnterCustomerPortal(Portal{}, false, nil)).To(BeTrue())
		g.Expect(MayEnterCustomerPortal(customer, true, nil)).To(BeTrue())
	})

	t.Run("partner members enter only the customers assigned to their partner", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnterCustomerPortal(partner, false, &partnerID)).To(BeTrue())
		g.Expect(MayEnterCustomerPortal(partner, false, &otherPartnerID)).To(BeFalse())
		g.Expect(MayEnterCustomerPortal(partner, false, nil)).To(BeFalse())
	})

	t.Run("customer members enter no other customer", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnterCustomerPortal(customer, false, nil)).To(BeFalse())
		g.Expect(MayEnterCustomerPortal(customer, false, &partnerID)).To(BeFalse())
	})
}

func TestMayEnterPartnerPortal(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Portal{PartnerOrganizationID: &partnerID}
	customer := Portal{CustomerOrganizationID: &customerID}

	g := NewWithT(t)
	g.Expect(MayEnterPartnerPortal(Portal{}, false)).To(BeTrue())
	g.Expect(MayEnterPartnerPortal(customer, true)).To(BeTrue())
	g.Expect(MayEnterPartnerPortal(partner, false)).To(BeFalse())
	g.Expect(MayEnterPartnerPortal(customer, false)).To(BeFalse())
}

func TestEnterLoadedPortal(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Portal{PartnerOrganizationID: &partnerID}
	customer := Portal{CustomerOrganizationID: &customerID}
	customerOrg := &types.CustomerOrganization{ID: customerID, PartnerOrganizationID: &partnerID}

	t.Run("an existing portal the session may enter is entered", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(enterLoadedPortal(partner, false, customer, customerOrg, false)).To(Equal(customerOrg))
		g.Expect(enterLoadedPortal(Portal{}, false, partner, nil, true)).To(BeNil())
	})

	t.Run("a portal missing from the organization is forbidden", func(t *testing.T) {
		g := NewWithT(t)
		_, err := enterLoadedPortal(Portal{}, true, customer, nil, true)
		g.Expect(err).To(MatchError(ErrPortalForbidden))
		_, err = enterLoadedPortal(Portal{}, true, partner, customerOrg, false)
		g.Expect(err).To(MatchError(ErrPortalForbidden))
	})

	t.Run("a portal naming both a customer and a partner is forbidden", func(t *testing.T) {
		g := NewWithT(t)
		both := Portal{CustomerOrganizationID: &customerID, PartnerOrganizationID: &partnerID}
		_, err := enterLoadedPortal(Portal{}, true, both, customerOrg, true)
		g.Expect(err).To(MatchError(ErrPortalForbidden))
	})
}

func TestIsOwnPortal(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Portal{PartnerOrganizationID: &partnerID}
	customer := Portal{CustomerOrganizationID: &customerID}

	t.Run("no portal and the membership's own are a regular session", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(IsOwnPortal(partner, Portal{})).To(BeTrue())
		g.Expect(IsOwnPortal(customer, Portal{CustomerOrganizationID: new(customerID)})).To(BeTrue())
	})

	t.Run("any other portal is not", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(IsOwnPortal(Portal{}, customer)).To(BeFalse())
		g.Expect(IsOwnPortal(partner, customer)).To(BeFalse())
		g.Expect(IsOwnPortal(customer, Portal{CustomerOrganizationID: new(uuid.New())})).To(BeFalse())
	})
}
