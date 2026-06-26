package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func DeploymentRevisionToAPI(r types.DeploymentRevisionWithCreator) *api.DeploymentRevisionResponse {
	response := &api.DeploymentRevisionResponse{
		ID:                     r.ID,
		CreatedAt:              r.CreatedAt,
		ApplicationVersionID:   r.ApplicationVersionID,
		ApplicationVersionName: r.ApplicationVersionName,
		ReleaseName:            r.ReleaseName,
		DockerType:             r.DockerType,
		ValuesYaml:             r.ValuesYaml,
		EnvFileData:            r.EnvFileData,
		ForceRestart:           r.ForceRestart,
		IgnoreRevisionSkew:     r.IgnoreRevisionSkew,
		HelmOptions:            r.HelmOptions,
	}

	if r.CreatedByEmail != nil {
		creator := &api.DeploymentRevisionCreator{
			Email:                  *r.CreatedByEmail,
			CustomerOrganizationID: r.CreatedByCustomerOrganizationID,
			PartnerOrganizationID:  r.CreatedByPartnerOrganizationID,
		}
		if r.CreatedByName != nil {
			creator.Name = *r.CreatedByName
		}
		response.CreatedBy = creator
	}

	return response
}
