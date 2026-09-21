package types

import (
	"fmt"
	"slices"

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
		parsedVersion, err := semver.NewVersion(version.Name)
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

func compareApplicationVersionsByDate(a, b ApplicationVersion) int {
	return a.CreatedAt.Compare(b.CreatedAt)
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
		parsedVersion, err := semver.NewVersion(version.Name)
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
