package lens

import (
	"context"
	"errors"

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

func Own(membership types.OrganizationWithUserRole) Lens {
	return Lens{
		CustomerOrganizationID: membership.CustomerOrganizationID,
		PartnerOrganizationID:  membership.PartnerOrganizationID,
	}
}

func Customer(customerOrgID uuid.UUID) Lens {
	return Lens{CustomerOrganizationID: &customerOrgID}
}

func (l Lens) isVendor() bool {
	return l.CustomerOrganizationID == nil && l.PartnerOrganizationID == nil
}

var ErrForbidden = errors.New("you are not allowed to switch to this organization")

// IsOwn reports whether a request for the lens of the customer organization requestedCustomerOrgID stays in the own
// lens, which a request for none does as well.
func IsOwn(own Lens, requestedCustomerOrgID *uuid.UUID) bool {
	return requestedCustomerOrgID == nil || util.PtrEq(requestedCustomerOrgID, own.CustomerOrganizationID)
}

// Enter decides whether a request whose own lens is own may look through the lens of the customer organization
// customerOrgID, and returns that customer organization. A customer organization that does not belong to the
// organization is forbidden like one the request may not enter.
func Enter(
	ctx context.Context,
	orgID uuid.UUID,
	own Lens,
	isSuperAdmin bool,
	customerOrgID uuid.UUID,
) (*types.CustomerOrganization, error) {
	if customerOrg, err := db.GetCustomerOrganization(ctx, customerOrgID, orgID); errors.Is(err, distrerr.ErrNotFound) {
		return nil, ErrForbidden
	} else if err != nil {
		return nil, err
	} else if !MayEnterCustomer(own, isSuperAdmin, customerOrg.PartnerOrganizationID) {
		return nil, ErrForbidden
	} else {
		return customerOrg, nil
	}
}

func MayEnterCustomer(own Lens, isSuperAdmin bool, customerPartnerOrgID *uuid.UUID) bool {
	switch {
	case isSuperAdmin, own.isVendor():
		return true
	case own.PartnerOrganizationID != nil:
		return util.PtrEq(customerPartnerOrgID, own.PartnerOrganizationID)
	default:
		return false
	}
}
