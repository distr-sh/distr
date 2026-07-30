// Package advisory holds the business rules for security advisories that are independent
// of storage, most importantly the predicate deciding which advisories a customer
// organization may see.
package advisory

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// VersionRef identifies a single application or artifact version affected by a
// advisory, together with the application or artifact it belongs to.
//
// For artifacts the caller is expected to expand each affected version into every version
// that resolves to the same content, so that an entitlement pinning a different tag or a
// different manifest of the same multi-arch index still matches.
type VersionRef struct {
	VersionID uuid.UUID
	ParentID  uuid.UUID
}

// EntitlementRef is a customer's entitlement to an application or artifact.
// An empty VersionIDs covers every version of the parent, matching the behaviour of
// ApplicationEntitlement rows without ApplicationEntitlement_ApplicationVersion children and
// of ArtifactEntitlement_Artifact rows with a NULL artifact_version_id.
// A nil ExpiresAt never expires.
type EntitlementRef struct {
	ParentID   uuid.UUID
	VersionIDs []uuid.UUID
	ExpiresAt  *time.Time
}

func (e EntitlementRef) coversVersion(ref VersionRef, now time.Time) bool {
	if e.ParentID != ref.ParentID {
		return false
	}
	if e.ExpiresAt != nil && !e.ExpiresAt.After(now) {
		return false
	}
	if len(e.VersionIDs) == 0 {
		return true
	}
	for _, versionID := range e.VersionIDs {
		if versionID == ref.VersionID {
			return true
		}
	}
	return false
}

// CustomerView is everything about a single customer organization that the visibility rule
// depends on.
//
// The two OrgHas* flags describe the vendor organization as a whole and drive the fallback
// for vendors who do not use the licensing feature. They are deliberately tracked per
// entitlement kind: a vendor may gate artifacts but not applications, and in that case the
// application side must still fall back to "visible to everyone" while the artifact side
// stays gated.
type CustomerView struct {
	OrgHasApplicationEntitlements bool
	OrgHasArtifactEntitlements    bool
	ApplicationEntitlements       []EntitlementRef
	ArtifactEntitlements          []EntitlementRef
	// DeployedApplicationVersionIDs are the application versions the customer has deployed at
	// any point, including ones they have since upgraded away from.
	DeployedApplicationVersionIDs []uuid.UUID
}

func (v CustomerView) hasDeployedAny(affected []VersionRef) bool {
	return containsAnyVersion(affected, v.DeployedApplicationVersionIDs)
}

func containsAnyVersion(refs []VersionRef, versionIDs []uuid.UUID) bool {
	for _, ref := range refs {
		for _, versionID := range versionIDs {
			if versionID == ref.VersionID {
				return true
			}
		}
	}
	return false
}

// MarkedVersions are the versions an advisory marks, already expanded to every version that
// resolves to the same content.
//
// There are no fixed application versions because the application side is decided by what a
// deployment runs right now, which needs no notion of a fix: a deployment that has moved off
// every affected version is no longer exposed, whether or not the version it moved to is one
// the vendor marked.
type MarkedVersions struct {
	AffectedApplicationVersions []VersionRef
	AffectedArtifactVersions    []VersionRef
	FixedArtifactVersions       []VersionRef
}

// Exposure is what a customer or partner has actually done with the vendor's software.
type Exposure struct {
	// CurrentApplicationVersionIDs are the versions their deployments run right now, one per
	// deployment. Not every version they ever ran: that distinction is the whole point, since
	// visibility deliberately keeps the advisory in front of customers who have already
	// upgraded, and this is what then tells them they are in the clear.
	CurrentApplicationVersionIDs []uuid.UUID
	// PulledArtifactVersionIDs are every artifact version they have pulled from the registry.
	PulledArtifactVersionIDs []uuid.UUID
}

// IsStillAffected reports whether an advisory is a live problem for whoever is asking, which
// is what customers and partners are shown in place of the editorial status.
//
// A deployment counts while it still runs an affected version, which mirrors
// AdvisoryImpactStateAffected in GetAdvisoryImpactedDeployments so that the badge and the
// impact table cannot disagree.
//
// A pull cannot be observed the same way, because Distr sees that an artifact was downloaded
// but never what became of it. Pulling a version carrying the fix is the closest thing to
// evidence that the fix was taken, so it is what clears the artifact.
func IsStillAffected(marked MarkedVersions, exposure Exposure) bool {
	if containsAnyVersion(marked.AffectedApplicationVersions, exposure.CurrentApplicationVersionIDs) {
		return true
	}
	return hasUnfixedPull(marked, exposure.PulledArtifactVersionIDs)
}

// hasUnfixedPull reports whether an affected version of some artifact was pulled without a
// version of that same artifact carrying the fix also being pulled.
//
// The check is per artifact rather than across the advisory as a whole: an advisory covering
// two artifacts is not settled by upgrading one of them.
func hasUnfixedPull(marked MarkedVersions, pulledIDs []uuid.UUID) bool {
	pulled := make(map[uuid.UUID]struct{}, len(pulledIDs))
	for _, id := range pulledIDs {
		pulled[id] = struct{}{}
	}

	fixTaken := make(map[uuid.UUID]struct{})
	for _, ref := range marked.FixedArtifactVersions {
		if _, ok := pulled[ref.VersionID]; ok {
			fixTaken[ref.ParentID] = struct{}{}
		}
	}

	for _, ref := range marked.AffectedArtifactVersions {
		if _, ok := pulled[ref.VersionID]; !ok {
			continue
		}
		if _, ok := fixTaken[ref.ParentID]; ok {
			continue
		}
		return true
	}
	return false
}

// IsVisibleToCustomer reports whether a customer organization may see an advisory.
//
// The rule is:
//
//   - Only published and resolved advisories are ever customer visible.
//   - An advisory without an affected version is not customer visible. A vendor must
//     explicitly identify what is affected before publishing can disclose the advisory.
//   - It is visible when the customer has deployed at least one affected application version.
//     Deployments do not require an entitlement, so this is the only signal that keeps the
//     customer's view in sync with the vendor's deployment-based impact report. Artifacts need
//     no equivalent, because a gated artifact cannot be pulled without an entitlement anyway.
//   - Otherwise it is visible when the customer is entitled to at least one affected version.
//     Only affected versions grant visibility; versions recorded as fixed do not.
//   - Per entitlement kind, a vendor who has configured no entitlements of that kind at all
//     exposes every affected version of that kind to every customer. This mirrors
//     CheckEntitlementForArtifact so that vendors who do not use licensing still reach their
//     customers, and it is scoped per kind so that using one kind does not silently expose
//     the other.
//
// Callers must pass only affected versions, already scoped to the vendor organization.
func IsVisibleToCustomer(
	status types.AdvisoryStatus,
	affectedApplicationVersions []VersionRef,
	affectedArtifactVersions []VersionRef,
	view CustomerView,
	now time.Time,
) bool {
	if !status.IsCustomerVisible() {
		return false
	}

	if len(affectedApplicationVersions) == 0 && len(affectedArtifactVersions) == 0 {
		return false
	}

	if view.hasDeployedAny(affectedApplicationVersions) {
		return true
	}

	if isEntitledToAny(
		affectedApplicationVersions,
		view.ApplicationEntitlements,
		view.OrgHasApplicationEntitlements,
		now,
	) {
		return true
	}

	return isEntitledToAny(
		affectedArtifactVersions,
		view.ArtifactEntitlements,
		view.OrgHasArtifactEntitlements,
		now,
	)
}

func isEntitledToAny(
	affected []VersionRef,
	entitlements []EntitlementRef,
	orgHasEntitlements bool,
	now time.Time,
) bool {
	if len(affected) == 0 {
		return false
	}
	if !orgHasEntitlements {
		return true
	}
	for _, ref := range affected {
		for _, entitlement := range entitlements {
			if entitlement.coversVersion(ref, now) {
				return true
			}
		}
	}
	return false
}
