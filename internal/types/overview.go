package types

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type OverviewDeployment struct {
	DeploymentWithLatestRevision
	// CreatedByName and CreatedByImageID are only set when the revision was created by a member of the
	// customer organization, since a customer must not learn who works for the vendor.
	CreatedByName    *string    `db:"created_by_name"`
	CreatedByImageID *uuid.UUID `db:"created_by_image_id"`
}

type OverviewChecklist struct {
	ProfileCompleted   bool `db:"profile_completed"`
	TeamMembersInvited bool `db:"team_members_invited"`
	DeploymentCreated  bool `db:"deployment_created"`
	SecretCreated      bool `db:"secret_created"`
	AccessTokenCreated bool `db:"access_token_created"`
	ArtifactPulled     bool `db:"artifact_pulled"`
}

// OverviewLicense is the license key or entitlement that decides the state of a customer's license.
type OverviewLicense struct {
	Name string `db:"name"`
	// NotBefore is only set for a license key, since an entitlement is valid from its creation.
	NotBefore *time.Time `db:"not_before"`
	ExpiresAt *time.Time `db:"expires_at"`
}

type OverviewAlertSummary struct {
	AlertConfigurationCount int64 `db:"alert_configuration_count"`
	NotificationCount       int64 `db:"notification_count"`
}

type OverviewSupportBundle struct {
	ID        uuid.UUID           `db:"id"`
	Title     string              `db:"title"`
	Status    SupportBundleStatus `db:"status"`
	UpdatedAt time.Time           `db:"updated_at"`
	// UserName and UserImageID are only set for a member of the customer organization, like the creator
	// of an OverviewDeployment.
	UserName    *string    `db:"user_name"`
	UserImageID *uuid.UUID `db:"user_image_id"`
}

// OverviewActivityFilter selects the categories of events a customer organization is shown, which have to
// follow the features that gate the corresponding sections.
type OverviewActivityFilter struct {
	Deployments    bool
	Artifacts      bool
	Licensing      bool
	SupportBundles bool
	// EntitledApplicationsOnly and EntitledArtifactsOnly restrict versions to those the customer is
	// entitled to, which mirrors the visibility of the application and artifact lists.
	EntitledApplicationsOnly bool
	EntitledArtifactsOnly    bool
}

type OverviewActivityType string

const (
	OverviewActivityTypeApplicationVersionCreated  OverviewActivityType = "application_version_created"
	OverviewActivityTypeArtifactVersionCreated     OverviewActivityType = "artifact_version_created"
	OverviewActivityTypeArtifactVersionFirstPulled OverviewActivityType = "artifact_version_first_pulled"
	OverviewActivityTypeDeploymentRevisionCreated  OverviewActivityType = "deployment_revision_created"
	OverviewActivityTypeLicenseKeyCreated          OverviewActivityType = "license_key_created"
	OverviewActivityTypeLicenseKeyUpdated          OverviewActivityType = "license_key_updated"
	OverviewActivityTypeSupportBundleCreated       OverviewActivityType = "support_bundle_created"
	OverviewActivityTypeSupportBundleUploaded      OverviewActivityType = "support_bundle_uploaded"
	OverviewActivityTypeSupportBundleResolved      OverviewActivityType = "support_bundle_resolved"
	OverviewActivityTypeSupportBundleCanceled      OverviewActivityType = "support_bundle_canceled"
)

type OverviewActivity struct {
	ID        uuid.UUID            `db:"id"`
	CreatedAt time.Time            `db:"created_at"`
	Type      OverviewActivityType `db:"type"`
	// ResourceID is the application, artifact, deployment target, license key or support bundle the
	// event is about, depending on its type.
	ResourceID  uuid.UUID  `db:"resource_id"`
	Subject     string     `db:"subject"`
	Detail      *string    `db:"detail"`
	UserName    *string    `db:"user_name"`
	UserImageID *uuid.UUID `db:"user_image_id"`
}

// LatestDeploymentApplicationVersion returns the newest version of the application that a deployment can
// be updated to. Entitlements are nil when they do not restrict the customer, which makes every version a
// candidate. Otherwise, a deployment bound to an entitlement is limited to that one, since the binding
// cannot be changed, and an unbound deployment to the union of the customer's entitlements for the
// application, any of which it can still be bound to. An expired or missing entitlement covers nothing
// and one without versions covers every version. The ordering is derived from all versions of the
// application, see ApplicationVersionComparator.
func LatestDeploymentApplicationVersion(
	application Application,
	deploymentEntitlementID *uuid.UUID,
	entitlements []ApplicationEntitlement,
	now time.Time,
) *ApplicationVersion {
	compare := ApplicationVersionComparator(application.VersioningStrategy, application.Versions)
	if entitlements == nil {
		return LatestApplicationVersion(compare, application.Versions)
	}
	versionIDs := make(map[uuid.UUID]struct{})
	for _, entitlement := range entitlements {
		if entitlement.ApplicationID != application.ID ||
			(deploymentEntitlementID != nil && entitlement.ID != *deploymentEntitlementID) ||
			(entitlement.ExpiresAt != nil && !entitlement.ExpiresAt.After(now)) {
			continue
		} else if len(entitlement.Versions) == 0 {
			return LatestApplicationVersion(compare, application.Versions)
		}
		for _, version := range entitlement.Versions {
			versionIDs[version.ID] = struct{}{}
		}
	}
	candidates := slices.DeleteFunc(slices.Clone(application.Versions), func(v ApplicationVersion) bool {
		_, ok := versionIDs[v.ID]
		return !ok
	})
	return LatestApplicationVersion(compare, candidates)
}

// ApplicationUpdateAvailable reports whether latest is newer than the version with currentVersionID.
func ApplicationUpdateAvailable(application Application, currentVersionID uuid.UUID, latest *ApplicationVersion) bool {
	if latest == nil || latest.ID == currentVersionID {
		return false
	}
	index := slices.IndexFunc(application.Versions, func(v ApplicationVersion) bool { return v.ID == currentVersionID })
	if index < 0 {
		return true
	}
	compare := ApplicationVersionComparator(application.VersioningStrategy, application.Versions)
	return compare(*latest, application.Versions[index]) > 0
}

// NextExpiresAt returns the next expiration among the credentials of the token that still authenticate,
// or nil when none of them expires. Once every credential has expired, it returns the last expiration.
// A credential that expired while another one still authenticates is ignored as a completed rotation.
func (tok AccessToken) NextExpiresAt(now time.Time) *time.Time {
	var expirations []*time.Time
	if tok.KeyIsCredential {
		expirations = append(expirations, tok.ExpiresAt)
	}
	for _, slot := range AccessTokenSecretSlots {
		if secret := tok.Secret(slot); secret != nil {
			expirations = append(expirations, secret.ExpiresAt)
		}
	}
	var next, lastExpired *time.Time
	valid := false
	for _, expiresAt := range expirations {
		switch {
		case expiresAt == nil:
			valid = true
		case expiresAt.After(now):
			valid = true
			if next == nil || expiresAt.Before(*next) {
				next = expiresAt
			}
		case lastExpired == nil || expiresAt.After(*lastExpired):
			lastExpired = expiresAt
		}
	}
	if valid {
		return next
	}
	return lastExpired
}
