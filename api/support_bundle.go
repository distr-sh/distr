package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

// Configuration

type SupportBundleConfigurationEnvVar struct {
	ID       *uuid.UUID `json:"id,omitempty"`
	Name     string     `json:"name"`
	Redacted bool       `json:"redacted"`
}

type SupportBundleConfiguration struct {
	ID        uuid.UUID                          `json:"id"`
	CreatedAt time.Time                          `json:"createdAt"`
	EnvVars   []SupportBundleConfigurationEnvVar `json:"envVars"`
}

type CreateUpdateSupportBundleConfigurationRequest struct {
	EnvVars []SupportBundleConfigurationEnvVar `json:"envVars"`
}

func (r *CreateUpdateSupportBundleConfigurationRequest) Validate() error {
	seen := make(map[string]struct{}, len(r.EnvVars))
	for _, ev := range r.EnvVars {
		key := strings.ToLower(strings.TrimSpace(ev.Name))
		if _, exists := seen[key]; exists {
			return validation.NewValidationFailedError(
				fmt.Sprintf("duplicate environment variable name: %v", ev.Name))
		}
		seen[key] = struct{}{}
	}
	return nil
}

// Bundle

type SupportBundle struct {
	ID                       uuid.UUID `json:"id"`
	CreatedAt                time.Time `json:"createdAt"`
	CustomerOrganizationID   uuid.UUID `json:"customerOrganizationId"`
	CustomerOrganizationName string    `json:"customerOrganizationName"`
	CreatedByUserAccountID   uuid.UUID `json:"createdByUserAccountId"`
	CreatedByUserName        string    `json:"createdByUserName"`
	CreatedByImageURL        *string   `json:"createdByImageUrl,omitempty"`
	Title                    *string   `json:"title,omitempty"`
	Description              *string   `json:"description,omitempty"`
	Status                   string    `json:"status"`
	ResourceCount            int64     `json:"resourceCount"`
}

type SupportBundleDetail struct {
	SupportBundle
	Resources []SupportBundleResource `json:"resources"`
	Comments  []SupportBundleComment  `json:"comments"`
}

type CreateSupportBundleRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type CreateSupportBundleResponse struct {
	SupportBundle
	CollectCommand string `json:"collectCommand"`
}

type UpdateSupportBundleStatusRequest struct {
	Status string `json:"status"`
}

// Resource

type SupportBundleResource struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
}

type CreateSupportBundleResourceRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Comment

type SupportBundleComment struct {
	ID            uuid.UUID `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UserAccountID uuid.UUID `json:"userAccountId"`
	UserName      string    `json:"userName"`
	UserImageURL  *string   `json:"userImageUrl,omitempty"`
	Content       string    `json:"content"`
}

type CreateSupportBundleCommentRequest struct {
	Content string `json:"content"`
}
