package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type DeploymentRevisionCreator struct {
	ID                     *uuid.UUID `json:"id,omitempty"`
	Name                   string     `json:"name,omitempty"`
	Email                  string     `json:"email,omitempty"`
	ImageID                *uuid.UUID `json:"imageId,omitempty"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
	PartnerOrganizationID  *uuid.UUID `json:"partnerOrganizationId,omitempty"`
	Deleted                bool       `json:"deleted,omitempty"`
}

type DeploymentRevisionResponse struct {
	ID                     uuid.UUID                  `json:"id"`
	CreatedAt              time.Time                  `json:"createdAt"`
	ApplicationVersionID   uuid.UUID                  `json:"applicationVersionId"`
	ApplicationVersionName string                     `json:"applicationVersionName"`
	ReleaseName            *string                    `json:"releaseName,omitempty"`
	DockerType             *types.DockerType          `json:"dockerType,omitempty"`
	ValuesYaml             []byte                     `json:"valuesYaml,omitempty"`
	EnvFileData            []byte                     `json:"envFileData,omitempty"`
	ForceRestart           bool                       `json:"forceRestart"`
	IgnoreRevisionSkew     bool                       `json:"ignoreRevisionSkew"`
	HelmOptions            *types.HelmOptions         `json:"helmOptions,omitempty"`
	CreatedBy              *DeploymentRevisionCreator `json:"createdBy,omitempty"`
}
