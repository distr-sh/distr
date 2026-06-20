package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func EnvironmentToAPI(environment types.Environment) api.Environment {
	return api.Environment{
		ID:                  environment.ID,
		CreatedAt:           environment.CreatedAt,
		UpdatedAt:           environment.UpdatedAt,
		Name:                environment.Name,
		Description:         environment.Description,
		SortOrder:           environment.SortOrder,
		IsProduction:        environment.IsProduction,
		AllowDynamicTargets: environment.AllowDynamicTargets,
		RetentionPolicyID:   environment.RetentionPolicyID,
	}
}
