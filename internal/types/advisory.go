package types

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidAdvisoryStatus   = fmt.Errorf("invalid advisory status")
	ErrInvalidAdvisorySeverity = fmt.Errorf("invalid advisory severity")
)

type AdvisoryStatus string

const (
	AdvisoryStatusTriage    AdvisoryStatus = "triage"
	AdvisoryStatusDraft     AdvisoryStatus = "draft"
	AdvisoryStatusPublished AdvisoryStatus = "published"
	AdvisoryStatusResolved  AdvisoryStatus = "resolved"
	AdvisoryStatusCanceled  AdvisoryStatus = "canceled"
)

func ParseAdvisoryStatus(value string) (AdvisoryStatus, error) {
	switch AdvisoryStatus(value) {
	case AdvisoryStatusTriage:
		return AdvisoryStatusTriage, nil
	case AdvisoryStatusDraft:
		return AdvisoryStatusDraft, nil
	case AdvisoryStatusPublished:
		return AdvisoryStatusPublished, nil
	case AdvisoryStatusResolved:
		return AdvisoryStatusResolved, nil
	case AdvisoryStatusCanceled:
		return AdvisoryStatusCanceled, nil
	default:
		return "", fmt.Errorf("%w: %v", ErrInvalidAdvisoryStatus, value)
	}
}

func (s AdvisoryStatus) IsCustomerVisible() bool {
	return s == AdvisoryStatusPublished || s == AdvisoryStatusResolved
}

type AdvisorySeverity string

const (
	AdvisorySeverityNone     AdvisorySeverity = "none"
	AdvisorySeverityLow      AdvisorySeverity = "low"
	AdvisorySeverityMedium   AdvisorySeverity = "medium"
	AdvisorySeverityHigh     AdvisorySeverity = "high"
	AdvisorySeverityCritical AdvisorySeverity = "critical"
)

func ParseAdvisorySeverity(value string) (AdvisorySeverity, error) {
	switch AdvisorySeverity(value) {
	case AdvisorySeverityNone:
		return AdvisorySeverityNone, nil
	case AdvisorySeverityLow:
		return AdvisorySeverityLow, nil
	case AdvisorySeverityMedium:
		return AdvisorySeverityMedium, nil
	case AdvisorySeverityHigh:
		return AdvisorySeverityHigh, nil
	case AdvisorySeverityCritical:
		return AdvisorySeverityCritical, nil
	default:
		return "", fmt.Errorf("%w: %v", ErrInvalidAdvisorySeverity, value)
	}
}

type AdvisoryVersionRelation string

const (
	AdvisoryVersionRelationAffected AdvisoryVersionRelation = "affected"
	AdvisoryVersionRelationPatched  AdvisoryVersionRelation = "patched"
)

type AdvisoryEventType string

const (
	AdvisoryEventTypePublished     AdvisoryEventType = "published"
	AdvisoryEventTypeStatusChanged AdvisoryEventType = "status_changed"
	AdvisoryEventTypeEdited        AdvisoryEventType = "edited"
	AdvisoryEventTypeComment       AdvisoryEventType = "comment"
)

type Advisory struct {
	ID                     uuid.UUID        `db:"id"`
	CreatedAt              time.Time        `db:"created_at"`
	UpdatedAt              time.Time        `db:"updated_at"`
	OrganizationID         uuid.UUID        `db:"organization_id"`
	CreatedByUserAccountID *uuid.UUID       `db:"created_by_user_account_id"`
	Title                  string           `db:"title"`
	Description            string           `db:"description"`
	Status                 AdvisoryStatus   `db:"status"`
	Severity               AdvisorySeverity `db:"severity"`
	CveID                  *string          `db:"cve_id"`
	PublishedAt            *time.Time       `db:"published_at"`
	ResolvedAt             *time.Time       `db:"resolved_at"`
}

type AdvisoryWithDetails struct {
	Advisory
	CreatedByUserName    *string    `db:"created_by_user_name"`
	CreatedByImageID     *uuid.UUID `db:"created_by_image_id"`
	Tags                 []string   `db:"tags"`
	AffectedVersionCount int64      `db:"affected_version_count"`
	PatchedVersionCount  int64      `db:"patched_version_count"`
	ReferenceCount       int64      `db:"reference_count"`
	// CallerAffected reports whether the requesting customer or partner is still exposed,
	// either through a deployment or a pull they never patched. Nil for vendors, who see the
	// status instead. Stamped by applyScope rather than selected, hence not a column.
	CallerAffected *bool `db:"-"`
}

type AdvisoryReference struct {
	ID         uuid.UUID `db:"id"`
	AdvisoryID uuid.UUID `db:"advisory_id"`
	URL        string    `db:"url"`
	Label      *string   `db:"label"`
}

type AdvisoryApplicationVersion struct {
	AdvisoryID             uuid.UUID               `db:"advisory_id"`
	ApplicationID          uuid.UUID               `db:"application_id"`
	ApplicationName        string                  `db:"application_name"`
	ApplicationType        DeploymentType          `db:"application_type"`
	ApplicationImageID     *uuid.UUID              `db:"application_image_id"`
	ApplicationVersionID   uuid.UUID               `db:"application_version_id"`
	ApplicationVersionName string                  `db:"application_version_name"`
	Relation               AdvisoryVersionRelation `db:"relation"`
}

type AdvisoryArtifactVersion struct {
	AdvisoryID      uuid.UUID  `db:"advisory_id"`
	ArtifactID      uuid.UUID  `db:"artifact_id"`
	ArtifactName    string     `db:"artifact_name"`
	ArtifactImageID *uuid.UUID `db:"artifact_image_id"`
	// ArtifactVersionName is a digest when the vendor marked the version by digest rather than
	// by tag, in which case ArtifactVersionTags holds the tags pointing at the same content.
	ArtifactVersionID     uuid.UUID               `db:"artifact_version_id"`
	ArtifactVersionName   string                  `db:"artifact_version_name"`
	ArtifactVersionDigest string                  `db:"artifact_version_digest"`
	ArtifactVersionTags   []string                `db:"artifact_version_tags"`
	Relation              AdvisoryVersionRelation `db:"relation"`
}

type AdvisoryEvent struct {
	ID            uuid.UUID         `db:"id"`
	CreatedAt     time.Time         `db:"created_at"`
	AdvisoryID    uuid.UUID         `db:"advisory_id"`
	UserAccountID *uuid.UUID        `db:"user_account_id"`
	Type          AdvisoryEventType `db:"type"`
	Message       *string           `db:"message"`
}

type AdvisoryEventWithUser struct {
	AdvisoryEvent
	UserName    *string    `db:"user_name"`
	UserImageID *uuid.UUID `db:"user_image_id"`
}

// AdvisoryImpactState is derived from the version a deployment's current revision runs, per
// query rather than stored, so it always reflects the versions currently marked affected and
// patched. There is deliberately no state expressing that a version is older or newer than an
// affected one: application versions have no authoritative ordering in the schema, so the
// vendor's own affected and patched markings are the only sound signal.
type AdvisoryImpactState string

const (
	AdvisoryImpactStateAffected AdvisoryImpactState = "affected"
	AdvisoryImpactStatePatched  AdvisoryImpactState = "patched"
	// AdvisoryImpactStateNotAffected means the deployment ran an affected version at some point
	// but has since moved to a version marked neither affected nor patched.
	AdvisoryImpactStateNotAffected AdvisoryImpactState = "not_affected"
)

// AdvisoryImpactedDeployment is one deployment that has run an affected application
// version at some point. There is exactly one row per deployment: ApplicationVersion* is the
// most recent affected version it ran and LastDeployedAt is when, while
// CurrentApplicationVersion* is what it runs today and is what State is derived from.
type AdvisoryImpactedDeployment struct {
	CustomerOrganizationID        *uuid.UUID          `db:"customer_organization_id"`
	CustomerOrganizationName      *string             `db:"customer_organization_name"`
	DeploymentID                  uuid.UUID           `db:"deployment_id"`
	DeploymentTargetID            uuid.UUID           `db:"deployment_target_id"`
	DeploymentTargetName          string              `db:"deployment_target_name"`
	ApplicationID                 uuid.UUID           `db:"application_id"`
	ApplicationName               string              `db:"application_name"`
	ApplicationVersionID          uuid.UUID           `db:"application_version_id"`
	ApplicationVersionName        string              `db:"application_version_name"`
	CurrentApplicationVersionID   uuid.UUID           `db:"current_application_version_id"`
	CurrentApplicationVersionName string              `db:"current_application_version_name"`
	State                         AdvisoryImpactState `db:"state"`
	LastDeployedAt                time.Time           `db:"last_deployed_at"`
}

// AdvisoryImpactedPull aggregates registry pulls of an affected artifact version.
// This records who downloaded the artifact, which is not the same as who is running it:
// nothing in the schema links an ArtifactVersion to an ApplicationVersion.
type AdvisoryImpactedPull struct {
	CustomerOrganizationID   *uuid.UUID `db:"customer_organization_id"`
	CustomerOrganizationName *string    `db:"customer_organization_name"`
	ArtifactID               uuid.UUID  `db:"artifact_id"`
	ArtifactName             string     `db:"artifact_name"`
	ArtifactVersionID        uuid.UUID  `db:"artifact_version_id"`
	ArtifactVersionName      string     `db:"artifact_version_name"`
	PullCount                int64      `db:"pull_count"`
	LastPulledAt             time.Time  `db:"last_pulled_at"`
}
