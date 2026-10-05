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

func TestSortApplicationVersionsLegacyFallsBackToCreationDate(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	versions := []ApplicationVersion{
		version("2.0.0", base.Add(time.Minute)),
		version("1.0.0", base),
		version("nightly", base.Add(2*time.Minute)),
	}

	SortApplicationVersions(VersioningStrategyLegacy, versions)
	g.Expect([]string{versions[0].Name, versions[1].Name, versions[2].Name}).
		To(Equal([]string{"1.0.0", "2.0.0", "nightly"}))
}

func TestSortAdvisoryApplicationVersionsBySemverWithinApplication(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	appA, appB := uuid.New(), uuid.New()
	versions := []AdvisoryApplicationVersion{
		{
			ApplicationID: appB, ApplicationName: "Bravo",
			ApplicationVersionID: uuid.New(), ApplicationVersionName: "1.0.0",
			ApplicationVersioningStrategy: VersioningStrategySemver, ApplicationVersionCreatedAt: base,
		},
		{
			ApplicationID: appA, ApplicationName: "Alpha",
			ApplicationVersionID: uuid.New(), ApplicationVersionName: "1.31.0",
			ApplicationVersioningStrategy: VersioningStrategySemver, ApplicationVersionCreatedAt: base.Add(time.Minute),
		},
		{
			ApplicationID: appA, ApplicationName: "Alpha",
			ApplicationVersionID: uuid.New(), ApplicationVersionName: "1.31.1",
			ApplicationVersioningStrategy: VersioningStrategySemver, ApplicationVersionCreatedAt: base,
		},
	}

	SortAdvisoryApplicationVersions(versions)
	g.Expect(versions[0].ApplicationName).To(Equal("Alpha"))
	g.Expect(versions[0].ApplicationVersionName).To(Equal("1.31.0"))
	g.Expect(versions[1].ApplicationVersionName).To(Equal("1.31.1"))
	g.Expect(versions[2].ApplicationName).To(Equal("Bravo"))
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
