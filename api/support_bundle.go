package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

var envVarNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Configuration

type SupportBundleConfigurationEnvVar struct {
	Name     string `json:"name"`
	Redacted bool   `json:"redacted"`
}

type CreateUpdateSupportBundleConfigurationRequest struct {
	EnvVars []SupportBundleConfigurationEnvVar `json:"envVars"`
}

func (r *CreateUpdateSupportBundleConfigurationRequest) Validate() error {
	seen := make(map[string]struct{}, len(r.EnvVars))
	for i, ev := range r.EnvVars {
		name := strings.TrimSpace(ev.Name)
		if name == "" {
			return validation.NewValidationFailedError("environment variable name must not be empty")
		}
		if !envVarNamePattern.MatchString(name) {
			return validation.NewValidationFailedError(
				fmt.Sprintf("invalid environment variable name: %v (must match [A-Za-z_][A-Za-z0-9_]*)", ev.Name))
		}
		r.EnvVars[i].Name = name
		key := strings.ToLower(name)
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
	ID                       uuid.UUID  `json:"id"`
	CreatedAt                time.Time  `json:"createdAt"`
	CustomerOrganizationID   uuid.UUID  `json:"customerOrganizationId"`
	CustomerOrganizationName string     `json:"customerOrganizationName"`
	CreatedByUserAccountID   uuid.UUID  `json:"createdByUserAccountId"`
	CreatedByUserName        string     `json:"createdByUserName"`
	CreatedByImageURL        *string    `json:"createdByImageUrl,omitempty"`
	Title                    string     `json:"title"`
	Description              *string    `json:"description,omitempty"`
	Status                   string     `json:"status"`
	ResourceCount            int64      `json:"resourceCount"`
	CommentCount             int64      `json:"commentCount"`
	LastCommentAt            *time.Time `json:"lastCommentAt,omitempty"`
	StatusChangedByUserName  *string    `json:"statusChangedByUserName,omitempty"`
	StatusChangedByImageURL  *string    `json:"statusChangedByImageUrl,omitempty"`
	StatusChangedAt          *time.Time `json:"statusChangedAt,omitempty"`
}

type SupportBundleDetail struct {
	SupportBundle
	Resources      []SupportBundleResource `json:"resources"`
	Comments       []SupportBundleComment  `json:"comments"`
	CollectCommand *string                 `json:"collectCommand,omitempty"`
}

type CreateSupportBundleRequest struct {
	Title       string  `json:"title"`
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

type SupportBundleResourceSummary struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
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
