package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomEmailConfigurationToResponse(c types.CustomEmailConfiguration) api.CustomEmailConfiguration {
	return api.CustomEmailConfiguration{
		ID:              c.ID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
		OrganizationID:  c.OrganizationID,
		Enabled:         c.Enabled,
		FromAddress:     c.FromAddress,
		SMTPHost:        c.SMTPHost,
		SMTPPort:        c.SMTPPort,
		SMTPUsername:    c.SMTPUsername,
		SMTPPasswordSet: c.SMTPPassword != "",
		SMTPImplicitTLS: c.SMTPImplicitTLS,
	}
}
