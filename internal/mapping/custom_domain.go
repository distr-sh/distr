package mapping

import (
	"time"

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

func CustomDomainVerificationToAPI(
	customDomainID uuid.UUID,
	dnsVerified bool,
	dnsDetail string,
	dnsCheckedAt time.Time,
) api.CustomDomainVerification {
	return api.CustomDomainVerification{
		CustomDomainID: customDomainID,
		DNSVerified:    dnsVerified,
		DNSDetail:      dnsDetail,
		DNSCheckedAt:   dnsCheckedAt,
	}
}
