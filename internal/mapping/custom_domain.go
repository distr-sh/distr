package mapping

import (
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomDomainVerificationToAPI(
	domain types.CustomDomain,
	dnsVerified bool,
	dnsDetail string,
	dnsCheckedAt time.Time,
) api.CustomDomainVerification {
	return api.CustomDomainVerification{
		CustomDomainID: domain.ID,
		DNSVerified:    dnsVerified,
		DNSDetail:      dnsDetail,
		DNSCheckedAt:   dnsCheckedAt,
	}
}
