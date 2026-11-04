package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomerSettingsToAPI(
	customerOrganization types.CustomerOrganization,
	defaultControllerImageRepository string,
) api.CustomerSettingsResponse {
	return api.CustomerSettingsResponse{
		ControllerImageRepository:        customerOrganization.ControllerImageRepository,
		DefaultControllerImageRepository: defaultControllerImageRepository,
	}
}
