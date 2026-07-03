package customdomains

import (
	"net/mail"
	"regexp"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
)

var urlSchemeRegex = regexp.MustCompile("^https?://")

func AppDomainOrDefault(b *types.OrganizationBranding) string {
	if b != nil && b.AppDomain != nil {
		d := *b.AppDomain
		if urlSchemeRegex.MatchString(d) {
			return d
		} else {
			scheme := urlSchemeRegex.FindString(env.Host())
			if scheme == "" {
				scheme = "https://"
			}
			return scheme + d
		}
	} else {
		return env.Host()
	}
}

func RegistryDomainOrDefault(b *types.OrganizationBranding) string {
	if b != nil && b.RegistryDomain != nil {
		return *b.RegistryDomain
	} else {
		return env.RegistryHost()
	}
}

func EmailFromAddressParsedOrDefault(b *types.OrganizationBranding) (*mail.Address, error) {
	if b != nil && b.EmailFromAddress != nil {
		return mail.ParseAddress(*b.EmailFromAddress)
	} else {
		return util.PtrTo(env.GetMailerConfig().FromAddress), nil
	}
}
