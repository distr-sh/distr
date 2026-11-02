package lens

import (
	"context"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

// Lens is the part of an organization a request looks at:
//
//	CustomerOrganizationID | PartnerOrganizationID | Lens
//	nil                    | nil                   | the vendor
//	set                    | nil                   | a customer organization
//	nil                    | set                   | a partner organization
//
// Every request looks through a lens, the own one being that of its user's membership; a request is in a lens
// (IsLens) when it looks through the one of a customer organization its user is not a member of. Only the lens of a
// customer organization can be entered.
type Lens struct {
	CustomerOrganizationID *uuid.UUID
	PartnerOrganizationID  *uuid.UUID
}

func Own(org types.OrganizationWithUserRole) Lens {
	return Lens{
		CustomerOrganizationID: org.CustomerOrganizationID,
		PartnerOrganizationID:  org.PartnerOrganizationID,
	}
}

func Customer(customerOrgID uuid.UUID) Lens {
	return Lens{CustomerOrganizationID: &customerOrgID}
}

func (l Lens) isVendor() bool {
	return l.CustomerOrganizationID == nil && l.PartnerOrganizationID == nil
}

// IsOwn reports whether a request for the lens of the customer organization requestedCustomerOrgID stays in the own
// lens, which a request for none does as well.
func IsOwn(own Lens, requestedCustomerOrgID *uuid.UUID) bool {
	return requestedCustomerOrgID == nil || util.PtrEq(requestedCustomerOrgID, own.CustomerOrganizationID)
}

func GetCustomerOrganizationForLens(
	ctx context.Context,
	orgID uuid.UUID,
	own Lens,
	isSuperAdmin bool,
	customerOrgID uuid.UUID,
) (*types.CustomerOrganization, error) {
	if customerOrg, err := db.GetCustomerOrganization(ctx, customerOrgID, orgID); err != nil {
		return nil, err
	} else if !MayEnter(own, isSuperAdmin, customerOrg.PartnerOrganizationID) {
		return nil, distrerr.ErrNotFound
	} else {
		return customerOrg, nil
	}
}

func MayEnter(own Lens, isSuperAdmin bool, customerPartnerOrgID *uuid.UUID) bool {
	switch {
	case isSuperAdmin, own.isVendor():
		return true
	case own.PartnerOrganizationID != nil:
		return util.PtrEq(customerPartnerOrgID, own.PartnerOrganizationID)
	default:
		return false
	}
}
