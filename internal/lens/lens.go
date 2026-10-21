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

// Target is the part of an organization a session operates in: the vendor's when both IDs are nil, otherwise the
// one partner or customer organization that is set. A session operates in a lens when its target is not the one of
// its user's membership.
type Target struct {
	CustomerOrganizationID *uuid.UUID
	PartnerOrganizationID  *uuid.UUID
}

func MembershipTarget(membership types.OrganizationWithUserRole) Target {
	return Target{
		CustomerOrganizationID: membership.CustomerOrganizationID,
		PartnerOrganizationID:  membership.PartnerOrganizationID,
	}
}

func (t Target) isVendor() bool {
	return t.CustomerOrganizationID == nil && t.PartnerOrganizationID == nil
}

var ErrForbidden = errors.New("you are not allowed to switch to this organization")

func IsOwn(own, requested Target) bool {
	return requested.isVendor() ||
		(util.PtrEq(requested.CustomerOrganizationID, own.CustomerOrganizationID) &&
			util.PtrEq(requested.PartnerOrganizationID, own.PartnerOrganizationID))
}

// Enter decides whether a session whose membership has the target own may operate in the lens requested, and
// returns the customer organization of that lens, nil for a partner. A customer or partner organization that
// does not belong to the organization is forbidden like one the session may not enter.
func Enter(
	ctx context.Context,
	orgID uuid.UUID,
	own Target,
	isSuperAdmin bool,
	requested Target,
) (*types.CustomerOrganization, error) {
	var customerOrg *types.CustomerOrganization
	if requested.CustomerOrganizationID != nil {
		var err error
		if customerOrg, err = db.GetCustomerOrganization(
			ctx, *requested.CustomerOrganizationID, orgID,
		); errors.Is(err, distrerr.ErrNotFound) {
			return nil, ErrForbidden
		} else if err != nil {
			return nil, err
		}
	} else if requested.PartnerOrganizationID != nil {
		if err := db.ValidatePartnerOrgBelongsToOrg(
			ctx, *requested.PartnerOrganizationID, orgID,
		); errors.Is(err, db.ErrPartnerOrgNotInOrg) {
			return nil, ErrForbidden
		} else if err != nil {
			return nil, err
		}
	}
	return EnterLoaded(own, isSuperAdmin, requested, customerOrg)
}

// EnterLoaded is Enter for a caller that already made sure the requested customer or partner organization
// belongs to the organization and loaded customerOrg, the requested customer organization.
func EnterLoaded(
	own Target,
	isSuperAdmin bool,
	requested Target,
	customerOrg *types.CustomerOrganization,
) (*types.CustomerOrganization, error) {
	switch {
	case requested.CustomerOrganizationID != nil && requested.PartnerOrganizationID == nil:
		if customerOrg == nil || customerOrg.ID != *requested.CustomerOrganizationID {
			return nil, errors.New("the requested customer organization was not loaded")
		} else if MayEnterCustomer(own, isSuperAdmin, customerOrg.PartnerOrganizationID) {
			return customerOrg, nil
		}
	case requested.PartnerOrganizationID != nil && requested.CustomerOrganizationID == nil:
		if MayEnterPartner(own, isSuperAdmin) {
			return nil, nil
		}
	}
	return nil, ErrForbidden
}

func MayEnterCustomer(own Target, isSuperAdmin bool, customerPartnerOrgID *uuid.UUID) bool {
	switch {
	case isSuperAdmin, own.isVendor():
		return true
	case own.PartnerOrganizationID != nil:
		return util.PtrEq(customerPartnerOrgID, own.PartnerOrganizationID)
	default:
		return false
	}
}

func MayEnterPartner(own Target, isSuperAdmin bool) bool {
	return isSuperAdmin || own.isVendor()
}
