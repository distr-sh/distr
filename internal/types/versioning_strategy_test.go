package types

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func version(name string, createdAt time.Time) ApplicationVersion {
	return ApplicationVersion{ID: uuid.New(), Name: name, CreatedAt: createdAt}
}

func archived(v ApplicationVersion) ApplicationVersion {
	archivedAt := v.CreatedAt.Add(time.Hour)
	v.ArchivedAt = &archivedAt
	return v
}

func latestOf(strategy VersioningStrategy, versions []ApplicationVersion) *ApplicationVersion {
	return LatestApplicationVersion(ApplicationVersionComparator(strategy, versions), versions)
}

func TestLatestApplicationVersionSemver(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()

	// The newest version is neither the last created nor the lexicographically greatest.
	versions := []ApplicationVersion{
		version("1.0.0", base),
		version("1.10.0", base.Add(time.Minute)),
		version("1.9.0", base.Add(2*time.Minute)),
	}

	latest := latestOf(VersioningStrategySemver, versions)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("1.10.0"))

	// A prerelease precedes its own release but still follows every earlier one.
	withPrerelease := slices.Concat(versions, []ApplicationVersion{
		version("2.0.0-rc.1", base.Add(3*time.Minute)),
	})
	latest = latestOf(VersioningStrategySemver, withPrerelease)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("2.0.0-rc.1"))

	withRelease := slices.Concat(withPrerelease, []ApplicationVersion{
		version("2.0.0", base.Add(4*time.Minute)),
	})
	latest = latestOf(VersioningStrategySemver, withRelease)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("2.0.0"))

	g.Expect(latestOf(VersioningStrategySemver, nil)).To(BeNil())
}

func TestLatestApplicationVersionSkipsArchived(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	versions := []ApplicationVersion{
		version("1.0.0", base),
		archived(version("2.0.0", base.Add(time.Minute))),
	}

	latest := latestOf(VersioningStrategySemver, versions)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("1.0.0"))
}

func TestLatestApplicationVersionChronological(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	versions := []ApplicationVersion{
		version("2.0.0", base),
		version("1.0.0", base.Add(time.Minute)),
	}

	latest := latestOf(VersioningStrategyChronological, versions)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("1.0.0"), "the creation date decides, not the name")
}

func TestLatestApplicationVersionLegacyFallsBackToCreationDate(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()

	semverOnly := []ApplicationVersion{
		version("2.0.0", base),
		version("1.0.0", base.Add(time.Minute)),
	}
	latest := latestOf(VersioningStrategyLegacy, semverOnly)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("2.0.0"), "SemVer applies while every name parses")

	// One unparseable name makes the whole set fall back, not just the pair it takes part in.
	withUnparseable := slices.Concat(semverOnly, []ApplicationVersion{
		version("nightly", base.Add(2*time.Minute)),
	})
	latest = latestOf(VersioningStrategyLegacy, withUnparseable)
	g.Expect(latest).NotTo(BeNil())
	g.Expect(latest.Name).To(Equal("nightly"))
}

func TestApplicationVersionComparatorTreatsEquivalentVersionsAsEqual(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	prefixed := version("v1.0.0", base)
	plain := version("1.0.0", base.Add(time.Minute))
	build := version("1.0.0+build.2", base.Add(2*time.Minute))
	versions := []ApplicationVersion{prefixed, plain, build}

	compare := ApplicationVersionComparator(VersioningStrategySemver, versions)
	g.Expect(compare(prefixed, plain)).To(BeZero(), "the v prefix is not part of the version")
	g.Expect(compare(plain, build)).To(BeZero(), "build metadata does not affect precedence")
}

func names(versions []ApplicationVersion) []string {
	result := make([]string, len(versions))
	for i, v := range versions {
		result[i] = v.Name
	}
	return result
}

func versionsForSortTests(base time.Time) []ApplicationVersion {
	return []ApplicationVersion{
		version("1.0.0", base),
		version("1.10.0", base.Add(time.Minute)),
		version("1.9.0", base.Add(2*time.Minute)),
		version("1.31.1", base.Add(3*time.Minute)),
		version("1.31.0", base.Add(4*time.Minute)),
	}
}

func TestSortApplicationVersionsSemver(t *testing.T) {
	g := NewWithT(t)
	versions := versionsForSortTests(time.Now())

	SortApplicationVersions(VersioningStrategySemver, versions)
	g.Expect(names(versions)).To(Equal([]string{"1.0.0", "1.9.0", "1.10.0", "1.31.0", "1.31.1"}))
}

func TestSortApplicationVersionsChronological(t *testing.T) {
	g := NewWithT(t)
	versions := versionsForSortTests(time.Now())

	SortApplicationVersions(VersioningStrategyChronological, versions)
	g.Expect(names(versions)).To(Equal([]string{"1.0.0", "1.10.0", "1.9.0", "1.31.1", "1.31.0"}))
}

func TestSortApplicationVersionsLegacyUsesSemverWhenEveryNameParses(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	versions := []ApplicationVersion{
		version("2.0.0", base),
		version("1.0.0", base.Add(time.Minute)),
	}

	SortApplicationVersions(VersioningStrategyLegacy, versions)
	g.Expect(names(versions)).To(Equal([]string{"1.0.0", "2.0.0"}))
}

func TestSortApplicationVersionsLegacyFallsBackToCreationDate(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	versions := []ApplicationVersion{
		version("2.0.0", base.Add(time.Minute)),
		version("1.0.0", base),
		version("nightly", base.Add(2*time.Minute)),
	}

	SortApplicationVersions(VersioningStrategyLegacy, versions)
	g.Expect(names(versions)).To(Equal([]string{"1.0.0", "2.0.0", "nightly"}))
}

func advisoryVersion(
	applicationID uuid.UUID,
	applicationName string,
	strategy VersioningStrategy,
	name string,
	createdAt time.Time,
) AdvisoryApplicationVersion {
	return AdvisoryApplicationVersion{
		ApplicationID:                 applicationID,
		ApplicationName:               applicationName,
		ApplicationVersioningStrategy: strategy,
		ApplicationVersionID:          uuid.New(),
		ApplicationVersionName:        name,
		ApplicationVersionCreatedAt:   createdAt,
	}
}

func advisoryNames(versions []AdvisoryApplicationVersion) []string {
	result := make([]string, len(versions))
	for i, v := range versions {
		result[i] = v.ApplicationName + " " + v.ApplicationVersionName
	}
	return result
}

func TestSortAdvisoryApplicationVersionsBySemverWithinApplication(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	alpha, bravo := uuid.New(), uuid.New()
	versions := []AdvisoryApplicationVersion{
		advisoryVersion(bravo, "Bravo", VersioningStrategySemver, "1.0.0", base),
		advisoryVersion(alpha, "Alpha", VersioningStrategySemver, "1.31.0", base.Add(time.Minute)),
		advisoryVersion(alpha, "Alpha", VersioningStrategySemver, "1.31.1", base),
	}

	SortAdvisoryApplicationVersions(versions)
	g.Expect(advisoryNames(versions)).To(Equal([]string{"Alpha 1.31.0", "Alpha 1.31.1", "Bravo 1.0.0"}))
}

func TestSortAdvisoryApplicationVersionsByChronologicalWithinApplication(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	alpha, bravo := uuid.New(), uuid.New()
	versions := []AdvisoryApplicationVersion{
		advisoryVersion(bravo, "Bravo", VersioningStrategyChronological, "1.0.0", base),
		advisoryVersion(alpha, "Alpha", VersioningStrategyChronological, "2.0.0", base),
		advisoryVersion(alpha, "Alpha", VersioningStrategyChronological, "1.0.0", base.Add(time.Minute)),
	}

	SortAdvisoryApplicationVersions(versions)
	g.Expect(advisoryNames(versions)).To(Equal([]string{"Alpha 2.0.0", "Alpha 1.0.0", "Bravo 1.0.0"}))
}

func TestSortAdvisoryApplicationVersionsLegacyUsesSemverWhenEveryNameParses(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	app := uuid.New()
	versions := []AdvisoryApplicationVersion{
		advisoryVersion(app, "App", VersioningStrategyLegacy, "2.0.0", base),
		advisoryVersion(app, "App", VersioningStrategyLegacy, "1.0.0", base.Add(time.Minute)),
	}

	SortAdvisoryApplicationVersions(versions)
	g.Expect(advisoryNames(versions)).To(Equal([]string{"App 1.0.0", "App 2.0.0"}))
}

func TestSortAdvisoryApplicationVersionsLegacyFallsBackToCreationDate(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	app := uuid.New()
	versions := []AdvisoryApplicationVersion{
		advisoryVersion(app, "App", VersioningStrategyLegacy, "2.0.0", base),
		advisoryVersion(app, "App", VersioningStrategyLegacy, "1.0.0", base.Add(time.Minute)),
		advisoryVersion(app, "App", VersioningStrategyLegacy, "nightly", base.Add(2*time.Minute)),
	}

	SortAdvisoryApplicationVersions(versions)
	g.Expect(advisoryNames(versions)).To(Equal([]string{"App 2.0.0", "App 1.0.0", "App nightly"}))
}

func TestSortAdvisoryApplicationVersionsUsesEachApplicationsStrategy(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	alpha, bravo := uuid.New(), uuid.New()
	versions := []AdvisoryApplicationVersion{
		advisoryVersion(bravo, "Bravo", VersioningStrategySemver, "1.10.0", base),
		advisoryVersion(alpha, "Alpha", VersioningStrategyChronological, "2.0.0", base),
		advisoryVersion(bravo, "Bravo", VersioningStrategySemver, "1.9.0", base.Add(time.Minute)),
		advisoryVersion(alpha, "Alpha", VersioningStrategyChronological, "1.0.0", base.Add(time.Minute)),
	}

	SortAdvisoryApplicationVersions(versions)
	g.Expect(advisoryNames(versions)).To(Equal([]string{"Alpha 2.0.0", "Alpha 1.0.0", "Bravo 1.9.0", "Bravo 1.10.0"}))
}

func TestValidateVersionsForStrategy(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()

	g.Expect(ValidateVersionsForStrategy(VersioningStrategySemver, []ApplicationVersion{
		version("1.0.0", base),
		version("2.0.0-rc.1", base.Add(time.Minute)),
	})).To(Succeed())

	g.Expect(ValidateVersionsForStrategy(VersioningStrategySemver, []ApplicationVersion{
		version("1.0.0", base),
		version("nightly", base.Add(time.Minute)),
	})).To(MatchError(ContainSubstring(`"nightly" is not a valid SemVer version`)))

	g.Expect(ValidateVersionsForStrategy(VersioningStrategySemver, []ApplicationVersion{
		version("1.0.0", base),
		version("v1.0.0", base.Add(time.Minute)),
	})).To(MatchError(ContainSubstring(`"1.0.0" and "v1.0.0" are the same SemVer version`)))

	// An archived version still occupies its name, so it is validated too.
	g.Expect(ValidateVersionsForStrategy(VersioningStrategySemver, []ApplicationVersion{
		archived(version("1.0.0", base)),
		version("v1.0.0", base.Add(time.Minute)),
	})).NotTo(Succeed())

	// The other strategies impose no requirement on the names.
	for _, strategy := range []VersioningStrategy{VersioningStrategyChronological, VersioningStrategyLegacy} {
		g.Expect(ValidateVersionsForStrategy(strategy, []ApplicationVersion{
			version("nightly", base),
			version("nightly-2", base.Add(time.Minute)),
		})).To(Succeed(), "strategy %q", strategy)
	}
}
