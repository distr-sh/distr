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

func IsOwnPortal(own, requested Portal) bool {
	return requested.isVendor() ||
		(util.PtrEq(requested.CustomerOrganizationID, own.CustomerOrganizationID) &&
			util.PtrEq(requested.PartnerOrganizationID, own.PartnerOrganizationID))
}

func EnterPortal(
	ctx context.Context,
	orgID uuid.UUID,
	own Portal,
	isSuperAdmin bool,
	requested Portal,
) (*types.CustomerOrganization, error) {
	var customerOrg *types.CustomerOrganization
	var partnerOrgExists bool
	switch {
	case requested.CustomerOrganizationID != nil && requested.PartnerOrganizationID == nil:
		var err error
		customerOrg, err = db.GetCustomerOrganization(ctx, *requested.CustomerOrganizationID, orgID)
		if err != nil && !errors.Is(err, distrerr.ErrNotFound) {
			return nil, err
		}
	case requested.PartnerOrganizationID != nil && requested.CustomerOrganizationID == nil:
		err := db.ValidatePartnerOrgBelongsToOrg(ctx, *requested.PartnerOrganizationID, orgID)
		if err != nil && !errors.Is(err, db.ErrPartnerOrgNotInOrg) {
			return nil, err
		}
		partnerOrgExists = err == nil
	}
	return enterLoadedPortal(own, isSuperAdmin, requested, customerOrg, partnerOrgExists)
}

// enterLoadedPortal is EnterPortal for a caller that already looked up the requested portal: customerOrg is
// the requested customer organization and partnerOrgExists whether the requested partner organization
// belongs to the organization, both scoped to the organization the session operates in.
func enterLoadedPortal(
	own Portal,
	isSuperAdmin bool,
	requested Portal,
	customerOrg *types.CustomerOrganization,
	partnerOrgExists bool,
) (*types.CustomerOrganization, error) {
	switch {
	case requested.CustomerOrganizationID != nil && requested.PartnerOrganizationID == nil:
		if customerOrg != nil && MayEnterCustomerPortal(own, isSuperAdmin, customerOrg.PartnerOrganizationID) {
			return customerOrg, nil
		}
	case requested.PartnerOrganizationID != nil && requested.CustomerOrganizationID == nil:
		if partnerOrgExists && MayEnterPartnerPortal(own, isSuperAdmin) {
			return nil, nil
		}
	}
	return nil, ErrPortalForbidden
}

func MayEnterCustomerPortal(own Portal, isSuperAdmin bool, customerPartnerOrgID *uuid.UUID) bool {
	switch {
	case isSuperAdmin, own.isVendor():
		return true
	case own.PartnerOrganizationID != nil:
		return util.PtrEq(customerPartnerOrgID, own.PartnerOrganizationID)
	default:
		return false
	}
}

func MayEnterPartnerPortal(own Portal, isSuperAdmin bool) bool {
	return isSuperAdmin || own.isVendor()
}
