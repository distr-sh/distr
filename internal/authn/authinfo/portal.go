package authinfo

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

// Portal is the part of an organization a session operates in: the vendor portal when both IDs are nil,
// otherwise the portal of the one partner or customer organization that is set.
type Portal struct {
	CustomerOrganizationID *uuid.UUID
	PartnerOrganizationID  *uuid.UUID
}

func (p Portal) isVendor() bool {
	return p.CustomerOrganizationID == nil && p.PartnerOrganizationID == nil
}

var ErrPortalForbidden = errors.New("you are not allowed to switch to this organization")

// IsOwnPortal reports whether the requested portal is the membership's own, which a request for no portal at
// all stands for as well. Only a request for another portal puts a session into a foreign context, a portal its
// user is not a member of.
func IsOwnPortal(membership, requested Portal) bool {
	return requested.isVendor() ||
		(util.PtrEq(requested.CustomerOrganizationID, membership.CustomerOrganizationID) &&
			util.PtrEq(requested.PartnerOrganizationID, membership.PartnerOrganizationID))
}

// EnterPortal checks that a member of the organization may enter a portal that is not the membership's own
// and returns the customer organization of a customer portal. Vendor members and super admins may enter the
// portal of every customer and partner of the organization, partner members the portals of that partner's
// customers and customer members none. Anything else fails with ErrPortalForbidden.
//
// Authentication calls this on every request of a session in a foreign context, so that deleting the
// customer or partner, or assigning the customer to another partner, ends the session right away.
func EnterPortal(
	ctx context.Context,
	orgID uuid.UUID,
	membership Portal,
	isSuperAdmin bool,
	requested Portal,
) (*types.CustomerOrganization, error) {
	switch {
	case requested.CustomerOrganizationID != nil && requested.PartnerOrganizationID == nil:
		customerOrg, err := db.GetCustomerOrganization(ctx, *requested.CustomerOrganizationID, orgID)
		if errors.Is(err, distrerr.ErrNotFound) {
			return nil, ErrPortalForbidden
		} else if err != nil {
			return nil, err
		} else if !MayEnter(membership, isSuperAdmin, requested, customerOrg.PartnerOrganizationID) {
			return nil, ErrPortalForbidden
		}
		return customerOrg, nil
	case requested.PartnerOrganizationID != nil && requested.CustomerOrganizationID == nil:
		if !MayEnter(membership, isSuperAdmin, requested, nil) {
			return nil, ErrPortalForbidden
		}
		err := db.ValidatePartnerOrgBelongsToOrg(ctx, *requested.PartnerOrganizationID, orgID)
		if errors.Is(err, db.ErrPartnerOrgNotInOrg) {
			return nil, ErrPortalForbidden
		}
		return nil, err
	default:
		return nil, ErrPortalForbidden
	}
}

// MayEnter is the rule of EnterPortal for a portal known to exist in the organization, so that a list of portals
// to switch to offers exactly those EnterPortal accepts. customerPartnerOrgID is the partner the requested
// customer is assigned to.
func MayEnter(membership Portal, isSuperAdmin bool, requested Portal, customerPartnerOrgID *uuid.UUID) bool {
	switch {
	case isSuperAdmin, membership.isVendor():
		return true
	case membership.PartnerOrganizationID != nil:
		return requested.CustomerOrganizationID != nil &&
			util.PtrEq(customerPartnerOrgID, membership.PartnerOrganizationID)
	default:
		return false
	}
}
