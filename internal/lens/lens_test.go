package lens

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestMayEnterCustomer(t *testing.T) {
	partnerID, otherPartnerID, customerID := uuid.New(), uuid.New(), uuid.New()
	partner := Target{PartnerOrganizationID: &partnerID}
	customer := Target{CustomerOrganizationID: &customerID}

	t.Run("vendor members and super admins enter every customer", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnterCustomer(Target{}, false, &partnerID)).To(BeTrue())
		g.Expect(MayEnterCustomer(Target{}, false, nil)).To(BeTrue())
		g.Expect(MayEnterCustomer(customer, true, nil)).To(BeTrue())
	})

	t.Run("partner members enter only the customers assigned to their partner", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnterCustomer(partner, false, &partnerID)).To(BeTrue())
		g.Expect(MayEnterCustomer(partner, false, &otherPartnerID)).To(BeFalse())
		g.Expect(MayEnterCustomer(partner, false, nil)).To(BeFalse())
	})

	t.Run("customer members enter no other customer", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(MayEnterCustomer(customer, false, nil)).To(BeFalse())
		g.Expect(MayEnterCustomer(customer, false, &partnerID)).To(BeFalse())
	})
}

func TestMayEnterPartner(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Target{PartnerOrganizationID: &partnerID}
	customer := Target{CustomerOrganizationID: &customerID}

	g := NewWithT(t)
	g.Expect(MayEnterPartner(Target{}, false)).To(BeTrue())
	g.Expect(MayEnterPartner(customer, true)).To(BeTrue())
	g.Expect(MayEnterPartner(partner, false)).To(BeFalse())
	g.Expect(MayEnterPartner(customer, false)).To(BeFalse())
}

func TestEnterLoaded(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Target{PartnerOrganizationID: &partnerID}
	customer := Target{CustomerOrganizationID: &customerID}
	customerOrg := &types.CustomerOrganization{ID: customerID, PartnerOrganizationID: &partnerID}

	t.Run("a lens the session may enter is entered", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(EnterLoaded(partner, false, customer, customerOrg)).To(Equal(customerOrg))
		g.Expect(EnterLoaded(Target{}, false, partner, nil)).To(BeNil())
	})

	t.Run("a lens naming both a customer and a partner is forbidden", func(t *testing.T) {
		g := NewWithT(t)
		both := Target{CustomerOrganizationID: &customerID, PartnerOrganizationID: &partnerID}
		_, err := EnterLoaded(Target{}, true, both, customerOrg)
		g.Expect(err).To(MatchError(ErrForbidden))
	})
}

func TestIsOwn(t *testing.T) {
	partnerID, customerID := uuid.New(), uuid.New()
	partner := Target{PartnerOrganizationID: &partnerID}
	customer := Target{CustomerOrganizationID: &customerID}

	t.Run("no target and the membership's own are a regular session", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(IsOwn(partner, Target{})).To(BeTrue())
		g.Expect(IsOwn(customer, Target{CustomerOrganizationID: new(customerID)})).To(BeTrue())
	})

	t.Run("any other target is a lens", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(IsOwn(Target{}, customer)).To(BeFalse())
		g.Expect(IsOwn(partner, customer)).To(BeFalse())
		g.Expect(IsOwn(customer, Target{CustomerOrganizationID: new(uuid.New())})).To(BeFalse())
	})
}
