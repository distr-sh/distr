package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func DeploymentRevisionStatusToAPI(status types.DeploymentRevisionStatus) api.DeploymentRevisionStatus {
	return api.DeploymentRevisionStatus{
		ID:                   status.ID,
		CreatedAt:            status.CreatedAt,
		DeploymentRevisionID: status.DeploymentRevisionID,
		Type:                 string(status.Type),
		Message:              status.Message,
	}
}
