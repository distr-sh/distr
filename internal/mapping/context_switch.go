package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func ContextSwitchOrganizationToAPI(
	orgID uuid.UUID,
	logoImageID *uuid.UUID,
	customerOrgs []types.CustomerOrganization,
	partnerOrgs []types.PartnerOrganization,
) api.ContextSwitchOrganization {
	return api.ContextSwitchOrganization{
		ID:                    orgID,
		LogoURL:               CreatePublicImageURL(logoImageID),
		CustomerOrganizations: List(customerOrgs, CustomerOrganizationToContextSwitchPortal),
		PartnerOrganizations:  List(partnerOrgs, PartnerOrganizationToContextSwitchPortal),
	}
}

func CustomerOrganizationToContextSwitchPortal(customerOrg types.CustomerOrganization) api.ContextSwitchPortal {
	return api.ContextSwitchPortal{
		ID:       customerOrg.ID,
		Name:     customerOrg.Name,
		ImageURL: CreateImageURL(customerOrg.ImageID),
	}
}

func PartnerOrganizationToContextSwitchPortal(partnerOrg types.PartnerOrganization) api.ContextSwitchPortal {
	return api.ContextSwitchPortal{
		ID:   partnerOrg.ID,
		Name: partnerOrg.Name,
	}
}
