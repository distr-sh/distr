package lens

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestMayEnter(t *testing.T) {
	partnerID, otherPartnerID, customerID := uuid.New(), uuid.New(), uuid.New()
	partner := Lens{PartnerOrganizationID: &partnerID}
	customer := Customer(customerID)

	t.Run("vendor members and super admins enter every customer", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnter(Lens{}, false, &partnerID)).To(BeTrue())
		g.Expect(MayEnter(Lens{}, false, nil)).To(BeTrue())
		g.Expect(MayEnter(customer, true, nil)).To(BeTrue())
	})

	t.Run("partner members enter only the customers assigned to their partner", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnter(partner, false, &partnerID)).To(BeTrue())
		g.Expect(MayEnter(partner, false, &otherPartnerID)).To(BeFalse())
		g.Expect(MayEnter(partner, false, nil)).To(BeFalse())
	})

	t.Run("customer members enter no other customer", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnter(customer, false, nil)).To(BeFalse())
		g.Expect(MayEnter(customer, false, &partnerID)).To(BeFalse())
	})
}

func TestIsOwn(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Lens{PartnerOrganizationID: &partnerID}
	customer := Customer(customerID)

	t.Run("no lens and the membership's own are a regular request", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(IsOwn(partner, nil)).To(BeTrue())
		g.Expect(IsOwn(customer, new(customerID))).To(BeTrue())
	})

	t.Run("any other customer organization is a lens", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(IsOwn(Lens{}, &customerID)).To(BeFalse())
		g.Expect(IsOwn(partner, &customerID)).To(BeFalse())
		g.Expect(IsOwn(customer, new(uuid.New()))).To(BeFalse())
	})
}
