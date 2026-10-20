// Package advisory holds the business rules for security advisories that are independent
// of storage, most importantly the predicate deciding which advisories a customer
// organization may see.
package advisory

import (
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// VersionRef is a version together with the application or artifact it belongs to.
//
// For artifacts the caller is expected to expand each marked version into every version that
// resolves to the same content, so that an entitlement pinning a different tag or a different
// manifest of the same multi-arch index still matches.
type VersionRef struct {
	VersionID uuid.UUID
	ParentID  uuid.UUID
}

// EntitlementRef with an empty VersionIDs covers every version of the parent, matching the
// behaviour of ApplicationEntitlement rows without ApplicationEntitlement_ApplicationVersion
// children and of ArtifactEntitlement_Artifact rows with a NULL artifact_version_id.
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
	return slices.Contains(e.VersionIDs, ref.VersionID)
}

// CustomerView is what the visibility rule knows about one customer organization.
//
// The two OrgHas* flags describe the vendor organization as a whole and drive the fallback for
// vendors who do not use the licensing feature. They are tracked per entitlement kind because
// a vendor may gate artifacts but not applications, and the application side must then still
// fall back to "visible to everyone" while the artifact side stays gated.
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
	return slices.ContainsFunc(refs, func(ref VersionRef) bool {
		return slices.Contains(versionIDs, ref.VersionID)
	})
}

// MarkedVersions are the versions an advisory marks, already expanded to every version that
// resolves to the same content.
//
// There are no patched application versions because the application side is decided by what a
// deployment runs right now: a deployment that has moved off every affected version is no
// longer exposed, whether or not the version it moved to is one the vendor marked.
type MarkedVersions struct {
	AffectedApplicationVersions []VersionRef
	AffectedArtifactVersions    []VersionRef
	PatchedArtifactVersions     []VersionRef
}

// Exposure is what a customer or partner has actually done with the vendor's software.
type Exposure struct {
	// CurrentApplicationVersionIDs are the versions their deployments run right now, one per
	// deployment, not every version they ever ran. Visibility deliberately keeps an advisory in
	// front of customers who have already upgraded, and this is what then tells them they are
	// in the clear.
	CurrentApplicationVersionIDs []uuid.UUID
	PulledArtifactVersionIDs     []uuid.UUID
}

// IsStillAffected reports whether an advisory is a live problem for whoever is asking, which
// is what customers and partners are shown in place of the editorial status.
//
// A deployment counts while it still runs an affected version, which mirrors
// AdvisoryImpactStateAffected in GetAdvisoryImpactedDeployments so that the badge and the
// impact table cannot disagree.
//
// A pull cannot be observed the same way, because Distr sees that an artifact was downloaded
// but never what became of it. Pulling a patched version is the closest thing to evidence that
// the patch was taken, so it is what clears the artifact.
func IsStillAffected(marked MarkedVersions, exposure Exposure) bool {
	if containsAnyVersion(marked.AffectedApplicationVersions, exposure.CurrentApplicationVersionIDs) {
		return true
	}
	return hasUnpatchedPull(marked, exposure.PulledArtifactVersionIDs)
}

// hasUnpatchedPull checks each artifact on its own rather than the advisory as a whole, so that
// an advisory covering two artifacts is not settled by taking the patch for one of them.
func hasUnpatchedPull(marked MarkedVersions, pulledIDs []uuid.UUID) bool {
	wasPulled := func(ref VersionRef) bool {
		return slices.Contains(pulledIDs, ref.VersionID)
	}
	patchTaken := func(artifactID uuid.UUID) bool {
		return slices.ContainsFunc(marked.PatchedArtifactVersions, func(ref VersionRef) bool {
			return ref.ParentID == artifactID && wasPulled(ref)
		})
	}
	return slices.ContainsFunc(marked.AffectedArtifactVersions, func(ref VersionRef) bool {
		return wasPulled(ref) && !patchTaken(ref.ParentID)
	})
}

// VersionVisibility decides which of an advisory's marked versions a customer may be shown,
// for one kind of version: applications or artifacts.
//
// This is disclosure, not exposure. It never feeds IsStillAffected, because a customer running
// an affected version they were never entitled to is affected all the same.
type VersionVisibility struct {
	// OrgHasEntitlements reports whether the vendor gates this kind of version at all. A
	// vendor who does not discloses everything, mirroring IsVisibleToCustomer.
	OrgHasEntitlements bool
	Entitlements       []EntitlementRef
	// KnownVersionIDs are versions the customer demonstrably already has, by having deployed
	// or pulled them. They stay disclosed without an entitlement, because visibility itself is
	// granted by deployment: filtering them away would leave such a customer looking at an
	// advisory with nothing listed as affected.
	KnownVersionIDs []uuid.UUID
}

func (v VersionVisibility) Allows(ref VersionRef, now time.Time) bool {
	if !v.OrgHasEntitlements {
		return true
	}
	if slices.Contains(v.KnownVersionIDs, ref.VersionID) {
		return true
	}
	return slices.ContainsFunc(v.Entitlements, func(entitlement EntitlementRef) bool {
		return entitlement.coversVersion(ref, now)
	})
}

// FilterVisibleVersions keeps the rows whose version the customer may see, in order.
func FilterVisibleVersions[T any](
	rows []T, ref func(T) VersionRef, visibility VersionVisibility, now time.Time,
) []T {
	return slices.DeleteFunc(slices.Clone(rows), func(row T) bool {
		return !visibility.Allows(ref(row), now)
	})
}

// IsDisclosed is what a partner sees, and the customer-independent half of
// IsVisibleToCustomer. A vendor must identify what is affected before publishing can disclose
// an advisory.
func IsDisclosed(
	status types.AdvisoryStatus,
	affectedApplicationVersions []VersionRef,
	affectedArtifactVersions []VersionRef,
) bool {
	if !status.IsCustomerVisible() {
		return false
	}
	return len(affectedApplicationVersions) > 0 || len(affectedArtifactVersions) > 0
}

// IsVisibleToCustomer reports whether a customer organization may see an advisory. Beyond
// disclosure, three parts of the rule are not evident from the code:
//
//   - Having deployed an affected application version is enough. Deployments require no
//     entitlement, and this is the only signal that keeps the customer's view in sync with
//     the vendor's deployment-based impact report. Artifacts need no equivalent, because a
//     gated artifact cannot be pulled without an entitlement anyway.
//   - Only affected versions grant visibility. Versions recorded as patched do not.
//   - A vendor who has configured no entitlements of one kind exposes every affected version
//     of that kind to every customer, mirroring CheckEntitlementForArtifact so that vendors
//     who do not use licensing still reach their customers. Scoping that fallback per kind
//     keeps using one kind from silently exposing the other.
//
// Callers must pass only affected versions, already scoped to the vendor organization.
func IsVisibleToCustomer(
	status types.AdvisoryStatus,
	affectedApplicationVersions []VersionRef,
	affectedArtifactVersions []VersionRef,
	view CustomerView,
	now time.Time,
) bool {
	if !IsDisclosed(status, affectedApplicationVersions, affectedArtifactVersions) {
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
	return slices.ContainsFunc(affected, func(ref VersionRef) bool {
		return slices.ContainsFunc(entitlements, func(entitlement EntitlementRef) bool {
			return entitlement.coversVersion(ref, now)
		})
	})
}
