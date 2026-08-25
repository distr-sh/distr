package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func CustomDomainToAPI(domain types.CustomDomain) api.CustomDomain {
	return api.CustomDomain{
		ID:                     domain.ID,
		CreatedAt:              domain.CreatedAt,
		Domain:                 domain.Domain,
		DomainType:             domain.Type,
		OrganizationID:         domain.OrganizationID,
		CustomerOrganizationID: domain.CustomerOrganizationID,
		Verified:               domain.Verified(),
		VerifiedAt:             domain.VerifiedAt,
		VerificationCheckedAt:  domain.VerificationCheckedAt,
		VerificationError:      domain.VerificationError,
	}
}

func CustomDomainToInternal(
	request api.CreateCustomDomainRequest,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) types.CustomDomain {
	return types.CustomDomain{
		Domain:                 request.Domain,
		Type:                   request.DomainType,
		OrganizationID:         organizationID,
		CustomerOrganizationID: customerOrganizationID,
	}
}

func CustomDomainVerificationToAPI(domain types.CustomDomain, inconclusive bool) api.CustomDomainVerification {
	return api.CustomDomainVerification{
		CustomDomainID:        domain.ID,
		Verified:              domain.Verified(),
		VerifiedAt:            domain.VerifiedAt,
		VerificationCheckedAt: domain.VerificationCheckedAt,
		VerificationError:     domain.VerificationError,
		Inconclusive:          inconclusive,
	}
}
