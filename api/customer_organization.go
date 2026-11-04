package api

import (
	"fmt"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type CreateUpdateCustomerOrganizationRequest struct {
	Name     string                              `json:"name"`
	ImageID  *uuid.UUID                          `json:"imageId,omitempty"`
	Features []types.CustomerOrganizationFeature `json:"features,omitempty"`
}

var customerOrganizationFeatureDependencies = map[types.CustomerOrganizationFeature]types.CustomerOrganizationFeature{
	types.CustomerOrganizationFeatureAlerts:        types.CustomerOrganizationFeatureDeploymentTargets,
	types.CustomerOrganizationFeatureOidcProviders: types.CustomerOrganizationFeatureSettings,
}

func (r *CreateUpdateCustomerOrganizationRequest) Validate() error {
	for _, feature := range r.Features {
		if required, ok := customerOrganizationFeatureDependencies[feature]; ok && !slices.Contains(r.Features, required) {
			return validation.NewValidationFailedError(fmt.Sprintf("feature %v requires feature %v", feature, required))
		}
	}

	return nil
}

type CustomerOrganization struct {
	ID                    uuid.UUID                           `json:"id"`
	CreatedAt             time.Time                           `json:"createdAt"`
	Name                  string                              `json:"name"`
	ImageID               *uuid.UUID                          `json:"imageId,omitempty"`
	ImageURL              *string                             `json:"imageUrl,omitempty"`
	Features              []types.CustomerOrganizationFeature `json:"features"`
	PartnerOrganizationID *uuid.UUID                          `json:"partnerOrganizationId,omitempty"`
}

type CustomerOrganizationWithUsage struct {
	CustomerOrganization
	UserCount             int64 `json:"userCount"`
	DeploymentTargetCount int64 `json:"deploymentTargetCount"`
}
