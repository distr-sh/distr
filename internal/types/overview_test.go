package types

import (
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestLatestDeploymentApplicationVersion(t *testing.T) {
	g := NewWithT(t)
	now := time.Now()
	past := now.Add(-time.Hour)
	v2 := version("2.0.0", now)
	v1 := version("1.0.0", now.Add(time.Minute))
	nightly := version("nightly", now.Add(2*time.Minute))
	application := Application{
		ID:                 uuid.New(),
		VersioningStrategy: VersioningStrategyLegacy,
		Versions:           []ApplicationVersion{v2, v1, nightly},
	}
	entitlement := func(expiresAt *time.Time, versions ...ApplicationVersion) ApplicationEntitlement {
		return ApplicationEntitlement{ApplicationEntitlementWithVersions: ApplicationEntitlementWithVersions{
			ApplicationEntitlementBase: ApplicationEntitlementBase{
				ID: uuid.New(), ApplicationID: application.ID, ExpiresAt: expiresAt,
			},
			Versions: versions,
		}}
	}
	limited, all, expired := entitlement(nil, v2, v1), entitlement(nil), entitlement(&past)
	latestOf := func(deploymentEntitlementID *uuid.UUID, entitlements ...ApplicationEntitlement) *ApplicationVersion {
		return LatestDeploymentApplicationVersion(application, deploymentEntitlementID, entitlements, now)
	}

	// The unparseable name outside the entitlement still makes the legacy strategy order by date.
	g.Expect(latestOf(&limited.ID, limited, all)).To(HaveField("ID", v1.ID),
		"a bound deployment ignores the customer's other entitlements")
	g.Expect(latestOf(nil, limited, all)).To(HaveField("ID", nightly.ID),
		"an entitlement without versions covers every version")
	g.Expect(latestOf(&expired.ID, expired, all)).To(BeNil(), "an expired entitlement covers nothing")
	g.Expect(latestOf(new(uuid.New()), all)).To(BeNil(), "a missing entitlement covers nothing")
	g.Expect(latestOf(nil, expired)).To(BeNil())
	g.Expect(LatestDeploymentApplicationVersion(application, nil, nil, now)).To(HaveField("ID", nightly.ID),
		"without restricting entitlements every version is a candidate")
}

func TestApplicationUpdateAvailable(t *testing.T) {
	g := NewWithT(t)
	base := time.Now()
	v1 := version("1.0.0", base)
	v2 := version("2.0.0", base.Add(time.Minute))
	v3 := version("3.0.0", base.Add(2*time.Minute))
	application := Application{
		VersioningStrategy: VersioningStrategySemver,
		Versions:           []ApplicationVersion{v1, v2, v3},
	}

	g.Expect(ApplicationUpdateAvailable(application, v1.ID, &v2)).To(BeTrue())
	g.Expect(ApplicationUpdateAvailable(application, v2.ID, &v2)).To(BeFalse())
	g.Expect(ApplicationUpdateAvailable(application, v3.ID, &v2)).
		To(BeFalse(), "a deployment ahead of the latest entitled version has no update")
	g.Expect(ApplicationUpdateAvailable(application, v1.ID, nil)).To(BeFalse())
}

func TestAccessTokenNextExpiresAt(t *testing.T) {
	g := NewWithT(t)
	now := time.Now()
	past := now.Add(-48 * time.Hour)
	recently := now.Add(-time.Hour)
	soon := now.Add(time.Hour)
	later := now.Add(48 * time.Hour)

	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &later},
		Secret2: &AccessTokenSecret{ExpiresAt: &soon},
	}.NextExpiresAt(now)).To(Equal(&soon), "a secret expiring soon is reported although another one outlives it")

	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &soon},
		Secret2: &AccessTokenSecret{},
	}.NextExpiresAt(now)).To(Equal(&soon), "a secret without expiration does not hide one that expires")

	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &past},
		Secret2: &AccessTokenSecret{ExpiresAt: &later},
	}.NextExpiresAt(now)).To(Equal(&later), "a secret replaced by a rotation is ignored")

	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &past},
		Secret2: &AccessTokenSecret{},
	}.NextExpiresAt(now)).To(BeNil())

	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &past},
		Secret2: &AccessTokenSecret{ExpiresAt: &recently},
	}.NextExpiresAt(now)).To(Equal(&recently), "an expired token reports when its last credential expired")

	g.Expect(AccessToken{
		ExpiresAt: &soon,
		Secret1:   &AccessTokenSecret{ExpiresAt: &later},
	}.NextExpiresAt(now)).To(Equal(&later), "the expiration of a key that is no credential is ignored")
}
