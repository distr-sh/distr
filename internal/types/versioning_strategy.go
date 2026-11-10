package types

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
)

type VersioningStrategy string

const (
	VersioningStrategySemver        VersioningStrategy = "semver"
	VersioningStrategyChronological VersioningStrategy = "chronological"
	// VersioningStrategyLegacy orders versions the way Distr did before applications had a
	// strategy: by SemVer where every name parses, by creation date otherwise. It exists for
	// applications whose names were never validated and cannot be selected.
	VersioningStrategyLegacy VersioningStrategy = "legacy"
)

// SelectableVersioningStrategies are the strategies an application may be set to.
func SelectableVersioningStrategies() []VersioningStrategy {
	return []VersioningStrategy{VersioningStrategySemver, VersioningStrategyChronological}
}

func (s VersioningStrategy) IsSelectable() bool {
	return slices.Contains(SelectableVersioningStrategies(), s)
}

func IsSemverTag(name string) bool {
	_, err := parseSemverTag(name)
	return err == nil
}

func parseSemverTag(name string) (*semver.Version, error) {
	return semver.StrictNewVersion(strings.TrimPrefix(name, "v"))
}

// SortTaggedArtifactVersions orders versions by the strategy, newest first. Under semver, a version ranks by its
// highest SemVer tag, and the versions without one, like signatures, follow by creation date.
func SortTaggedArtifactVersions(strategy VersioningStrategy, versions []TaggedArtifactVersion) {
	highest := make(map[uuid.UUID]*semver.Version, len(versions))
	if strategy == VersioningStrategySemver {
		for _, version := range versions {
			for _, tag := range version.Tags {
				if parsed, err := parseSemverTag(tag.Name); err == nil && compareSemver(parsed, highest[version.ID]) > 0 {
					highest[version.ID] = parsed
				}
			}
		}
	}
	slices.SortStableFunc(versions, func(a, b TaggedArtifactVersion) int {
		return cmp.Or(compareSemver(highest[b.ID], highest[a.ID]), b.CreatedAt.Compare(a.CreatedAt))
	})
}

// CompareArtifactTags orders two release tags of an artifact by the strategy. Under semver, a tag that is not a SemVer
// version sorts before every SemVer one. The creation date decides where the strategy does not.
func CompareArtifactTags(strategy VersioningStrategy, a, b ArtifactVersion) int {
	if strategy == VersioningStrategySemver {
		parsedA, _ := parseSemverTag(a.Name)
		parsedB, _ := parseSemverTag(b.Name)
		if c := compareSemver(parsedA, parsedB); c != 0 {
			return c
		}
	}
	return a.CreatedAt.Compare(b.CreatedAt)
}

// ArtifactTagBehind reports whether a customer whose latest pull was the tag pulled has something to update to in
// the tag announced. A tag that points to the content they pulled already brings nothing new.
func ArtifactTagBehind(strategy VersioningStrategy, pulled, announced ArtifactVersion) bool {
	return pulled.ManifestBlobDigest != announced.ManifestBlobDigest &&
		CompareArtifactTags(strategy, pulled, announced) < 0
}

// compareSemver orders nil before every version.
func compareSemver(a, b *semver.Version) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	default:
		return a.Compare(b)
	}
}

// AllowsArtifactTag reports whether an artifact with the strategy may have a tag of the given name. Tags that are
// not releases, like signatures, are allowed under every strategy.
func (s VersioningStrategy) AllowsArtifactTag(name string) bool {
	return s != VersioningStrategySemver || !IsReleaseTag(name) || IsSemverTag(name)
}

func (s VersioningStrategy) AllowsAutomaticUpdates() bool {
	return s != VersioningStrategyLegacy
}

// ApplicationVersionComparator returns the ordering the strategy imposes on the given versions.
// It is derived from the whole set rather than from the two versions being compared, because the
// legacy strategy falls back to the creation date as soon as one name is not valid SemVer, and a
// comparator built from a subset would order a pair differently than the set it belongs to.
func ApplicationVersionComparator(
	strategy VersioningStrategy,
	versions []ApplicationVersion,
) func(a, b ApplicationVersion) int {
	if strategy == VersioningStrategyChronological {
		return compareApplicationVersionsByDate
	}
	parsed := make(map[uuid.UUID]*semver.Version, len(versions))
	for _, version := range versions {
		parsedVersion, err := parseSemver(version.Name)
		if err != nil && strategy == VersioningStrategyLegacy {
			return compareApplicationVersionsByDate
		} else if err == nil {
			parsed[version.ID] = parsedVersion
		}
	}
	return func(a, b ApplicationVersion) int {
		if parsedA, parsedB := parsed[a.ID], parsed[b.ID]; parsedA != nil && parsedB != nil {
			return parsedA.Compare(parsedB)
		}
		return compareApplicationVersionsByDate(a, b)
	}
}

// parseSemver rejects a name with a missing minor or patch number, which semver.NewVersion would
// complete with zeros.
func parseSemver(name string) (*semver.Version, error) {
	return semver.StrictNewVersion(strings.TrimPrefix(name, "v"))
}

func compareApplicationVersionsByDate(a, b ApplicationVersion) int {
	return a.CreatedAt.Compare(b.CreatedAt)
}

// SortApplicationVersions orders versions ASC by the application's versioning strategy. The
// comparator is built from the whole slice, matching ApplicationVersionComparator.
func SortApplicationVersions(strategy VersioningStrategy, versions []ApplicationVersion) {
	slices.SortStableFunc(versions, ApplicationVersionComparator(strategy, versions))
}

// SortAdvisoryApplicationVersions orders advisory version rows by application name, then by each
// application's versioning strategy (ASC). The comparator is built once per application from that
// application's marked versions.
func SortAdvisoryApplicationVersions(versions []AdvisoryApplicationVersion) {
	grouped := make(map[string][]AdvisoryApplicationVersion)
	for _, version := range versions {
		// NUL sorts before any character of a name (Postgres text cannot contain it), so a name that
		// is a prefix of another still sorts first.
		key := version.ApplicationName + "\x00" + version.ApplicationID.String()
		grouped[key] = append(grouped[key], version)
	}
	result := make([]AdvisoryApplicationVersion, 0, len(versions))
	for _, key := range slices.Sorted(maps.Keys(grouped)) {
		group := grouped[key]
		applicationVersions := make([]ApplicationVersion, len(group))
		for j, version := range group {
			applicationVersions[j] = version.ApplicationVersion
		}
		compare := ApplicationVersionComparator(group[0].ApplicationVersioningStrategy, applicationVersions)
		slices.SortStableFunc(group, func(a, b AdvisoryApplicationVersion) int {
			return compare(a.ApplicationVersion, b.ApplicationVersion)
		})
		result = append(result, group...)
	}
	copy(versions, result)
}

// LatestApplicationVersion returns the newest of the given versions, ignoring archived ones, or
// nil when none remain. The ordering comes from ApplicationVersionComparator, so that a subset
// (the versions an entitlement covers) is ordered by the ordering of the whole.
func LatestApplicationVersion(
	compare func(a, b ApplicationVersion) int,
	versions []ApplicationVersion,
) *ApplicationVersion {
	var latest *ApplicationVersion
	for i := range versions {
		if versions[i].ArchivedAt == nil && (latest == nil || compare(versions[i], *latest) > 0) {
			latest = &versions[i]
		}
	}
	return latest
}

// ValidateVersionsForStrategy reports whether the strategy can order the given versions. Archived
// versions are included, so that un-archiving one cannot invalidate the application. The returned
// error is written to the end user.
func ValidateVersionsForStrategy(strategy VersioningStrategy, versions []ApplicationVersion) error {
	if strategy != VersioningStrategySemver {
		return nil
	}
	parsed := make([]*semver.Version, 0, len(versions))
	for i, version := range versions {
		parsedVersion, err := parseSemver(version.Name)
		if err != nil {
			return fmt.Errorf("version %q is not a valid SemVer version", version.Name)
		}
		if j := slices.IndexFunc(parsed, func(other *semver.Version) bool {
			return parsedVersion.Compare(other) == 0
		}); j >= 0 {
			return fmt.Errorf("versions %q and %q are the same SemVer version",
				versions[j].Name, versions[i].Name)
		}
		parsed = append(parsed, parsedVersion)
	}
	return nil
}
