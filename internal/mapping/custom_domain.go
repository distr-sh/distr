package mapping

import (
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomDomainWithVerificationToAPI(
	domain types.CustomDomain,
	dnsVerified bool,
	dnsDetail string,
	dnsCheckedAt time.Time,
) api.CustomDomainWithVerification {
	return api.CustomDomainWithVerification{
		CustomDomain: domain,
		DNSVerified:  dnsVerified,
		DNSDetail:    dnsDetail,
		DNSCheckedAt: dnsCheckedAt,
	}
}
