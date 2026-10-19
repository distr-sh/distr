package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type OverviewUser struct {
	Name     string  `json:"name"`
	ImageURL *string `json:"imageUrl,omitempty"`
}

type OverviewDeploymentTarget struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Type        types.DeploymentType `json:"type"`
	Connected   bool                 `json:"connected"`
	Deployments []OverviewDeployment `json:"deployments"`
}

type OverviewDeployment struct {
	ID                            uuid.UUID                 `json:"id"`
	ApplicationID                 uuid.UUID                 `json:"applicationId"`
	ApplicationName               string                    `json:"applicationName"`
	ApplicationImageURL           *string                   `json:"applicationImageUrl,omitempty"`
	ApplicationTitleImageURL      *string                   `json:"applicationTitleImageUrl,omitempty"`
	ApplicationLink               string                    `json:"applicationLink,omitempty"`
	ApplicationVersionID          uuid.UUID                 `json:"applicationVersionId"`
	ApplicationVersionName        string                    `json:"applicationVersionName"`
	CurrentApplicationVersionID   *uuid.UUID                `json:"currentApplicationVersionId,omitempty"`
	CurrentApplicationVersionName *string                   `json:"currentApplicationVersionName,omitempty"`
	LatestApplicationVersionName  *string                   `json:"latestApplicationVersionName,omitempty"`
	UpdateAvailable               bool                      `json:"updateAvailable"`
	DeploymentRevisionID          uuid.UUID                 `json:"deploymentRevisionId"`
	DeploymentRevisionCreatedAt   time.Time                 `json:"deploymentRevisionCreatedAt"`
	DeploymentRevisionCreatedBy   *OverviewUser             `json:"deploymentRevisionCreatedBy,omitempty"`
	CurrentDeploymentRevisionID   *uuid.UUID                `json:"currentDeploymentRevisionId,omitempty"`
	LatestStatus                  *DeploymentRevisionStatus `json:"latestStatus,omitempty"`
	CurrentStatus                 *DeploymentRevisionStatus `json:"currentStatus,omitempty"`
}

type OverviewArtifactsResponse struct {
	Artifacts []OverviewArtifact `json:"artifacts"`
	HasPulled bool               `json:"hasPulled"`
}

type OverviewArtifact struct {
	ID                uuid.UUID             `json:"id"`
	Name              string                `json:"name"`
	ImageURL          *string               `json:"imageUrl,omitempty"`
	TitleImageURL     *string               `json:"titleImageUrl,omitempty"`
	LatestVersionName *string               `json:"latestVersionName,omitempty"`
	LastPull          *OverviewArtifactPull `json:"lastPull,omitempty"`
}

type OverviewArtifactPull struct {
	VersionName          string        `json:"versionName"`
	CreatedAt            time.Time     `json:"createdAt"`
	PulledBy             *OverviewUser `json:"pulledBy,omitempty"`
	DeploymentTargetName *string       `json:"deploymentTargetName,omitempty"`
	IsLatest             bool          `json:"isLatest"`
}

type OverviewChecklistResponse struct {
	ProfileCompleted   bool `json:"profileCompleted"`
	TeamMembersInvited bool `json:"teamMembersInvited"`
	DeploymentCreated  bool `json:"deploymentCreated"`
	SecretCreated      bool `json:"secretCreated"`
	AccessTokenCreated bool `json:"accessTokenCreated"`
	ArtifactPulled     bool `json:"artifactPulled"`
}

type OverviewLicenseResponse struct {
	Name      string     `json:"name"`
	NotBefore *time.Time `json:"notBefore,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type OverviewAlertsResponse struct {
	AlertConfigurationCount int64 `json:"alertConfigurationCount"`
	NotificationCount       int64 `json:"notificationCount"`
}

type OverviewAccessToken struct {
	ID         uuid.UUID  `json:"id"`
	Label      *string    `json:"label,omitempty"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

type OverviewSupportBundle struct {
	ID        uuid.UUID                 `json:"id"`
	Title     string                    `json:"title"`
	Status    types.SupportBundleStatus `json:"status"`
	UpdatedAt time.Time                 `json:"updatedAt"`
	UpdatedBy *OverviewUser             `json:"updatedBy,omitempty"`
}

type OverviewActivity struct {
	ID         uuid.UUID                  `json:"id"`
	CreatedAt  time.Time                  `json:"createdAt"`
	Type       types.OverviewActivityType `json:"type"`
	ResourceID uuid.UUID                  `json:"resourceId"`
	Subject    string                     `json:"subject"`
	Detail     *string                    `json:"detail,omitempty"`
	User       *OverviewUser              `json:"user,omitempty"`
}
