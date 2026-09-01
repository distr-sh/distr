package api

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

const (
	maxAdvisoryTitleLength = 200
	maxAdvisoryTagLength   = 50
	maxAdvisoryTagCount    = 20
)

type Advisory struct {
	ID                   uuid.UUID              `json:"id"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
	CreatedByUserName    *string                `json:"createdByUserName,omitempty"`
	CreatedByImageURL    *string                `json:"createdByImageUrl,omitempty"`
	Title                string                 `json:"title"`
	Status               types.AdvisoryStatus   `json:"status"`
	Severity             types.AdvisorySeverity `json:"severity"`
	CveID                *string                `json:"cveId,omitempty"`
	Tags                 []string               `json:"tags"`
	AffectedVersionCount int64                  `json:"affectedVersionCount"`
	FixedVersionCount    int64                  `json:"fixedVersionCount"`
	ReferenceCount       int64                  `json:"referenceCount"`
	PublishedAt          *time.Time             `json:"publishedAt,omitempty"`
	ResolvedAt           *time.Time             `json:"resolvedAt,omitempty"`
	// Affected reports whether the advisory is still a live problem for the requesting
	// customer or partner: a deployment of theirs runs an affected version, or they pulled an
	// affected artifact version without since pulling one that carries the fix. Omitted for
	// vendors, who see the status instead.
	Affected *bool `json:"affected,omitempty"`
}

type AdvisoryDetail struct {
	Advisory
	Description         string                       `json:"description"`
	References          []AdvisoryReference          `json:"references"`
	ApplicationVersions []AdvisoryApplicationVersion `json:"applicationVersions"`
	ArtifactVersions    []AdvisoryArtifactVersion    `json:"artifactVersions"`
	Events              []AdvisoryEvent              `json:"events"`
}

type AdvisoryReference struct {
	URL   string  `json:"url"`
	Label *string `json:"label,omitempty"`
}

type AdvisoryApplicationVersion struct {
	ApplicationID          uuid.UUID                     `json:"applicationId"`
	ApplicationName        string                        `json:"applicationName"`
	ApplicationType        types.DeploymentType          `json:"applicationType"`
	ApplicationImageURL    *string                       `json:"applicationImageUrl,omitempty"`
	ApplicationVersionID   uuid.UUID                     `json:"applicationVersionId"`
	ApplicationVersionName string                        `json:"applicationVersionName"`
	Relation               types.AdvisoryVersionRelation `json:"relation"`
}

type AdvisoryArtifactVersion struct {
	ArtifactID       uuid.UUID `json:"artifactId"`
	ArtifactName     string    `json:"artifactName"`
	ArtifactImageURL *string   `json:"artifactImageUrl,omitempty"`
	// ArtifactVersionName is a digest when the vendor marked the version by digest rather than
	// by tag, in which case ArtifactVersionTags holds the tags pointing at the same content.
	ArtifactVersionID     uuid.UUID                     `json:"artifactVersionId"`
	ArtifactVersionName   string                        `json:"artifactVersionName"`
	ArtifactVersionDigest string                        `json:"artifactVersionDigest"`
	ArtifactVersionTags   []string                      `json:"artifactVersionTags"`
	Relation              types.AdvisoryVersionRelation `json:"relation"`
}

type AdvisoryEvent struct {
	ID           uuid.UUID               `json:"id"`
	CreatedAt    time.Time               `json:"createdAt"`
	Type         types.AdvisoryEventType `json:"type"`
	Message      *string                 `json:"message,omitempty"`
	UserName     *string                 `json:"userName,omitempty"`
	UserImageURL *string                 `json:"userImageUrl,omitempty"`
}

// Impact

type AdvisoryImpact struct {
	Deployments []AdvisoryImpactedDeployment `json:"deployments"`
	Pulls       []AdvisoryImpactedPull       `json:"pulls"`
}

type AdvisoryImpactedDeployment struct {
	CustomerOrganizationID        *uuid.UUID                `json:"customerOrganizationId,omitempty"`
	CustomerOrganizationName      *string                   `json:"customerOrganizationName,omitempty"`
	DeploymentID                  uuid.UUID                 `json:"deploymentId"`
	DeploymentTargetID            uuid.UUID                 `json:"deploymentTargetId"`
	DeploymentTargetName          string                    `json:"deploymentTargetName"`
	ApplicationID                 uuid.UUID                 `json:"applicationId"`
	ApplicationName               string                    `json:"applicationName"`
	ApplicationVersionID          uuid.UUID                 `json:"applicationVersionId"`
	ApplicationVersionName        string                    `json:"applicationVersionName"`
	CurrentApplicationVersionID   uuid.UUID                 `json:"currentApplicationVersionId"`
	CurrentApplicationVersionName string                    `json:"currentApplicationVersionName"`
	State                         types.AdvisoryImpactState `json:"state"`
	LastDeployedAt                time.Time                 `json:"lastDeployedAt"`
}

type AdvisoryImpactedPull struct {
	CustomerOrganizationID   *uuid.UUID `json:"customerOrganizationId,omitempty"`
	CustomerOrganizationName *string    `json:"customerOrganizationName,omitempty"`
	ArtifactID               uuid.UUID  `json:"artifactId"`
	ArtifactName             string     `json:"artifactName"`
	ArtifactVersionID        uuid.UUID  `json:"artifactVersionId"`
	ArtifactVersionName      string     `json:"artifactVersionName"`
	PullCount                int64      `json:"pullCount"`
	LastPulledAt             time.Time  `json:"lastPulledAt"`
}

// Requests

type AdvisoryIDRequest struct {
	AdvisoryID uuid.UUID `path:"advisoryId"`
}

// ListAdvisoriesRequest holds the filters of the list endpoint. Each may be repeated to match
// any of several values, and omitting it entirely disables that filter.
type ListAdvisoriesRequest struct {
	Status   []string `query:"status"`
	Severity []string `query:"severity"`
	Tag      []string `query:"tag"`
}

type ParsedListAdvisoriesRequest struct {
	Statuses   []types.AdvisoryStatus
	Severities []types.AdvisorySeverity
	Tags       []string
}

func (r *ListAdvisoriesRequest) Parse() (ParsedListAdvisoriesRequest, error) {
	var parsed ParsedListAdvisoriesRequest

	for _, value := range r.Status {
		status, err := types.ParseAdvisoryStatus(value)
		if err != nil {
			return ParsedListAdvisoriesRequest{}, validation.NewValidationFailedError(err.Error())
		}
		if !slices.Contains(parsed.Statuses, status) {
			parsed.Statuses = append(parsed.Statuses, status)
		}
	}

	for _, value := range r.Severity {
		severity, err := types.ParseAdvisorySeverity(value)
		if err != nil {
			return ParsedListAdvisoriesRequest{}, validation.NewValidationFailedError(err.Error())
		}
		if !slices.Contains(parsed.Severities, severity) {
			parsed.Severities = append(parsed.Severities, severity)
		}
	}

	for _, value := range r.Tag {
		// A blank tag is dropped rather than rejected: it cannot match anything, and an empty
		// repeated query parameter is easy to produce by accident.
		if tag := strings.TrimSpace(value); tag != "" && !slices.Contains(parsed.Tags, tag) {
			parsed.Tags = append(parsed.Tags, tag)
		}
	}

	return parsed, nil
}

type CreateUpdateAdvisoryRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	// Status defaults to "triage" on create and leaves the status untouched when omitted on
	// update.
	Status types.AdvisoryStatus `json:"status,omitempty"`
	// CveID is unique per organization, ignoring case. Reusing one that another advisory
	// already carries is rejected with 409 Conflict.
	CveID                         *string             `json:"cveId"`
	Tags                          []string            `json:"tags"`
	References                    []AdvisoryReference `json:"references"`
	AffectedApplicationVersionIDs []uuid.UUID         `json:"affectedApplicationVersionIds"`
	FixedApplicationVersionIDs    []uuid.UUID         `json:"fixedApplicationVersionIds"`
	AffectedArtifactVersionIDs    []uuid.UUID         `json:"affectedArtifactVersionIds"`
	FixedArtifactVersionIDs       []uuid.UUID         `json:"fixedArtifactVersionIds"`
}

func (r *CreateUpdateAdvisoryRequest) Validate() error {
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return validation.NewValidationFailedError("title must not be empty")
	}
	if len(r.Title) > maxAdvisoryTitleLength {
		return validation.NewValidationFailedError(
			fmt.Sprintf("title must not be longer than %v characters", maxAdvisoryTitleLength))
	}

	if _, err := types.ParseAdvisorySeverity(r.Severity); err != nil {
		return validation.NewValidationFailedError(err.Error())
	}

	if r.Status != "" {
		status, err := types.ParseAdvisoryStatus(string(r.Status))
		if err != nil {
			return validation.NewValidationFailedError(err.Error())
		}
		r.Status = status
	}

	if r.CveID != nil {
		cveID := strings.TrimSpace(*r.CveID)
		if cveID == "" {
			r.CveID = nil
		} else {
			r.CveID = &cveID
		}
	}

	if err := r.validateTags(); err != nil {
		return err
	}
	if err := r.validateReferences(); err != nil {
		return err
	}
	return r.validateVersions()
}

func (r *CreateUpdateAdvisoryRequest) validateTags() error {
	if len(r.Tags) > maxAdvisoryTagCount {
		return validation.NewValidationFailedError(
			fmt.Sprintf("an advisory must not have more than %v tags", maxAdvisoryTagCount))
	}
	seen := make(map[string]struct{}, len(r.Tags))
	tags := make([]string, 0, len(r.Tags))
	for _, tag := range r.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return validation.NewValidationFailedError("tag must not be empty")
		}
		if len(tag) > maxAdvisoryTagLength {
			return validation.NewValidationFailedError(
				fmt.Sprintf("tag must not be longer than %v characters: %v", maxAdvisoryTagLength, tag))
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	r.Tags = tags
	return nil
}

func (r *CreateUpdateAdvisoryRequest) validateReferences() error {
	references := make([]AdvisoryReference, 0, len(r.References))
	for _, reference := range r.References {
		rawURL := strings.TrimSpace(reference.URL)
		if rawURL == "" {
			return validation.NewValidationFailedError("reference url must not be empty")
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return validation.NewValidationFailedError(
				fmt.Sprintf("reference url must be a valid http(s) url: %v", reference.URL))
		}
		if reference.Label != nil {
			label := strings.TrimSpace(*reference.Label)
			if label == "" {
				reference.Label = nil
			} else {
				reference.Label = &label
			}
		}
		reference.URL = rawURL
		references = append(references, reference)
	}
	r.References = references
	return nil
}

// validateVersions rejects a version that is listed as both affected and fixed, which the
// composite primary key on the link tables cannot represent.
func (r *CreateUpdateAdvisoryRequest) validateVersions() error {
	if overlap := firstOverlap(r.AffectedApplicationVersionIDs, r.FixedApplicationVersionIDs); overlap != nil {
		return validation.NewValidationFailedError(
			fmt.Sprintf("application version %v cannot be both affected and fixed", overlap))
	}
	if overlap := firstOverlap(r.AffectedArtifactVersionIDs, r.FixedArtifactVersionIDs); overlap != nil {
		return validation.NewValidationFailedError(
			fmt.Sprintf("artifact version %v cannot be both affected and fixed", overlap))
	}
	r.AffectedApplicationVersionIDs = deduplicate(r.AffectedApplicationVersionIDs)
	r.FixedApplicationVersionIDs = deduplicate(r.FixedApplicationVersionIDs)
	r.AffectedArtifactVersionIDs = deduplicate(r.AffectedArtifactVersionIDs)
	r.FixedArtifactVersionIDs = deduplicate(r.FixedArtifactVersionIDs)
	return nil
}

func firstOverlap(a, b []uuid.UUID) *uuid.UUID {
	inA := make(map[uuid.UUID]struct{}, len(a))
	for _, id := range a {
		inA[id] = struct{}{}
	}
	for _, id := range b {
		if _, exists := inA[id]; exists {
			return &id
		}
	}
	return nil
}

func deduplicate(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// PatchAdvisoryRequest is what the status and severity dropdowns send, where the rest of the
// advisory is not at hand. An omitted field is left as it is.
type PatchAdvisoryRequest struct {
	Status   *types.AdvisoryStatus   `json:"status,omitempty"`
	Severity *types.AdvisorySeverity `json:"severity,omitempty"`
}

func (r *PatchAdvisoryRequest) Validate() error {
	if r.Status == nil && r.Severity == nil {
		return validation.NewValidationFailedError("either status or severity must be given")
	}
	if r.Status != nil {
		status, err := types.ParseAdvisoryStatus(string(*r.Status))
		if err != nil {
			return validation.NewValidationFailedError(err.Error())
		}
		r.Status = &status
	}
	if r.Severity != nil {
		severity, err := types.ParseAdvisorySeverity(string(*r.Severity))
		if err != nil {
			return validation.NewValidationFailedError(err.Error())
		}
		r.Severity = &severity
	}
	return nil
}

type CreateAdvisoryCommentRequest struct {
	Content string `json:"content"`
}

func (r *CreateAdvisoryCommentRequest) Validate() error {
	r.Content = strings.TrimSpace(r.Content)
	if r.Content == "" {
		return validation.NewValidationFailedError("content must not be empty")
	}
	return nil
}
