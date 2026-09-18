package notification

import "github.com/distr-sh/distr/internal/types"

func customerOrganizationName(deploymentTarget types.DeploymentTargetFull) *string {
	if deploymentTarget.CustomerOrganization == nil {
		return nil
	}
	return &deploymentTarget.CustomerOrganization.Name
}
