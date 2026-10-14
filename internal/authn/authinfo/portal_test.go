package authinfo

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestMayEnter(t *testing.T) {
	partnerID, otherPartnerID, customerID := uuid.New(), uuid.New(), uuid.New()
	vendor := Portal{}
	partner := Portal{PartnerOrganizationID: &partnerID}
	customer := Portal{CustomerOrganizationID: &customerID}
	otherCustomer := Portal{CustomerOrganizationID: new(uuid.New())}
	otherPartner := Portal{PartnerOrganizationID: &otherPartnerID}

	t.Run("vendor members and super admins enter every customer and partner", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnter(vendor, false, otherCustomer, &partnerID)).To(BeTrue())
		g.Expect(MayEnter(vendor, false, otherPartner, nil)).To(BeTrue())
		g.Expect(MayEnter(customer, true, otherCustomer, nil)).To(BeTrue())
		g.Expect(MayEnter(customer, true, otherPartner, nil)).To(BeTrue())
	})

	t.Run("partner members enter only the customers assigned to their partner", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnter(partner, false, otherCustomer, &partnerID)).To(BeTrue())
		g.Expect(MayEnter(partner, false, otherCustomer, &otherPartnerID)).To(BeFalse())
		g.Expect(MayEnter(partner, false, otherCustomer, nil)).To(BeFalse())
		g.Expect(MayEnter(partner, false, otherPartner, nil)).To(BeFalse())
	})

	t.Run("customer members enter nothing", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnter(customer, false, otherCustomer, nil)).To(BeFalse())
		g.Expect(MayEnter(customer, false, otherCustomer, &partnerID)).To(BeFalse())
		g.Expect(MayEnter(customer, false, otherPartner, nil)).To(BeFalse())
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
