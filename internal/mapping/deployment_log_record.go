package mapping

import "github.com/distr-sh/distr/api"

func DeploymentLogRecordResourcesToAPI(active, archived []string) api.DeploymentLogRecordResourcesResponse {
	if active == nil {
		active = []string{}
	}
	if archived == nil {
		archived = []string{}
	}
	return api.DeploymentLogRecordResourcesResponse{
		Active:   active,
		Archived: archived,
	}
}
