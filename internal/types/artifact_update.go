package types

import (
	"slices"

	"github.com/google/uuid"
)

// CustomerArtifactVersion is a release tag of an artifact together with the customer organization it concerns,
// such as the tag the customer organization pulled last or one it may pull.
type CustomerArtifactVersion struct {
	CustomerOrganizationID uuid.UUID
	Artifact               Artifact
	ArtifactVersion        ArtifactVersion
}

// CustomerArtifactUpdate is the CustomerArtifactVersion a customer organization pulled last with the newer release
// tag that it may pull, nil when there is none.
type CustomerArtifactUpdate struct {
	CustomerArtifactVersion
	AvailableUpdate *ArtifactVersion
}

// ArtifactEntitlementGrant lets a customer organization pull the versions of an artifact with the manifest digest,
// or every version of it when ManifestBlobDigest is nil.
type ArtifactEntitlementGrant struct {
	CustomerOrganizationID uuid.UUID `db:"customer_organization_id"`
	ArtifactID             uuid.UUID `db:"artifact_id"`
	ManifestBlobDigest     *Digest   `db:"manifest_blob_digest"`
}

type customerArtifact struct {
	customerOrganizationID uuid.UUID
	artifactID             uuid.UUID
}

type customerArtifactDigest struct {
	customerArtifact
	digest Digest
}

// ArtifactTagVisibility decides which release tags a customer organization may pull.
type ArtifactTagVisibility struct {
	gated     bool
	artifacts map[customerArtifact]struct{}
	digests   map[customerArtifactDigest]struct{}
}

// NewArtifactTagVisibility restricts the artifacts that are not public to the grants when gated. Entitlements
// restrict a customer only once the vendor has created any.
func NewArtifactTagVisibility(gated bool, grants []ArtifactEntitlementGrant) ArtifactTagVisibility {
	visibility := ArtifactTagVisibility{
		gated:     gated,
		artifacts: map[customerArtifact]struct{}{},
		digests:   map[customerArtifactDigest]struct{}{},
	}
	for _, grant := range grants {
		key := customerArtifact{grant.CustomerOrganizationID, grant.ArtifactID}
		if grant.ManifestBlobDigest == nil {
			visibility.artifacts[key] = struct{}{}
		} else {
			visibility.digests[customerArtifactDigest{key, *grant.ManifestBlobDigest}] = struct{}{}
		}
	}
	return visibility
}

func (v ArtifactTagVisibility) Allows(customerOrganizationID uuid.UUID, artifact Artifact, tag ArtifactVersion) bool {
	if !v.gated || artifact.Public {
		return true
	}
	key := customerArtifact{customerOrganizationID, artifact.ID}
	if _, ok := v.artifacts[key]; ok {
		return true
	}
	_, ok := v.digests[customerArtifactDigest{key, tag.ManifestBlobDigest}]
	return ok
}

// AvailableArtifactUpdates adds to each pull the newest release tag of the artifact that the customer organization
// may pull, unless that tag is not newer than the one it pulled by the rule update notifications follow. The tags
// have to include every release tag of the pulled artifacts.
func AvailableArtifactUpdates(
	pulls []CustomerArtifactVersion,
	tags []ArtifactVersion,
	visibility ArtifactTagVisibility,
) []CustomerArtifactUpdate {
	tagsByArtifact := make(map[uuid.UUID][]ArtifactVersion)
	for _, tag := range tags {
		tagsByArtifact[tag.ArtifactID] = append(tagsByArtifact[tag.ArtifactID], tag)
	}
	sorted := make(map[uuid.UUID]struct{}, len(tagsByArtifact))

	result := make([]CustomerArtifactUpdate, len(pulls))
	for i, pull := range pulls {
		artifactTags := tagsByArtifact[pull.Artifact.ID]
		if _, ok := sorted[pull.Artifact.ID]; !ok {
			sortArtifactTags(pull.Artifact.VersioningStrategy, artifactTags)
			sorted[pull.Artifact.ID] = struct{}{}
		}
		index := slices.IndexFunc(artifactTags, func(tag ArtifactVersion) bool {
			return visibility.Allows(pull.CustomerOrganizationID, pull.Artifact, tag)
		})
		result[i] = CustomerArtifactUpdate{CustomerArtifactVersion: pull}
		if index >= 0 &&
			ArtifactTagBehind(pull.Artifact.VersioningStrategy, pull.ArtifactVersion, artifactTags[index]) {
			result[i].AvailableUpdate = &artifactTags[index]
		}
	}
	return result
}
