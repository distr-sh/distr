package advisory_test

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/advisory"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

var (
	now = time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	applicationID      = uuid.New()
	otherApplicationID = uuid.New()
	versionOneID       = uuid.New()
	versionTwoID       = uuid.New()

	artifactID      = uuid.New()
	otherArtifactID = uuid.New()
	amd64VersionID  = uuid.New()
	arm64VersionID  = uuid.New()
	indexVersionID  = uuid.New()
)

func appVersion(versionID uuid.UUID) advisory.VersionRef {
	return advisory.VersionRef{VersionID: versionID, ParentID: applicationID}
}

func artifactVersion(versionID uuid.UUID) advisory.VersionRef {
	return advisory.VersionRef{VersionID: versionID, ParentID: artifactID}
}

func TestIsVisibleToCustomerStatusGating(t *testing.T) {
	g := NewWithT(t)

	// A vendor with no entitlements at all, so status is the only thing under test.
	view := advisory.CustomerView{}
	affected := []advisory.VersionRef{appVersion(versionOneID)}

	cases := []struct {
		status  types.AdvisoryStatus
		visible bool
	}{
		{types.AdvisoryStatusTriage, false},
		{types.AdvisoryStatusDraft, false},
		{types.AdvisoryStatusPublished, true},
		{types.AdvisoryStatusResolved, true},
	}
	for _, tc := range cases {
		g.Expect(advisory.IsVisibleToCustomer(tc.status, affected, nil, view, now)).
			To(Equal(tc.visible), "status %q", tc.status)
	}
}

func TestIsVisibleToCustomerNoAffectedVersions(t *testing.T) {
	g := NewWithT(t)

	view := advisory.CustomerView{
		OrgHasApplicationEntitlements: true,
		OrgHasArtifactEntitlements:    true,
	}

	g.Expect(advisory.IsVisibleToCustomer(
		types.AdvisoryStatusPublished, nil, nil, view, now,
	)).To(BeFalse())

	g.Expect(advisory.IsVisibleToCustomer(
		types.AdvisoryStatusDraft, nil, nil, view, now,
	)).To(BeFalse())
}

func TestIsVisibleToCustomerApplicationEntitlements(t *testing.T) {
	cases := []struct {
		name     string
		affected []advisory.VersionRef
		view     advisory.CustomerView
		visible  bool
	}{
		{
			name:     "entitlement covering all versions",
			affected: []advisory.VersionRef{appVersion(versionOneID)},
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				ApplicationEntitlements: []advisory.EntitlementRef{
					{ParentID: applicationID},
				},
			},
			visible: true,
		},
		{
			name:     "entitlement pinned to the affected version",
			affected: []advisory.VersionRef{appVersion(versionOneID)},
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				ApplicationEntitlements: []advisory.EntitlementRef{
					{ParentID: applicationID, VersionIDs: []uuid.UUID{versionOneID}},
				},
			},
			visible: true,
		},
		{
			name:     "entitlement pinned to a different version",
			affected: []advisory.VersionRef{appVersion(versionOneID)},
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				ApplicationEntitlements: []advisory.EntitlementRef{
					{ParentID: applicationID, VersionIDs: []uuid.UUID{versionTwoID}},
				},
			},
			visible: false,
		},
		{
			name:     "entitlement for a different application",
			affected: []advisory.VersionRef{appVersion(versionOneID)},
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				ApplicationEntitlements: []advisory.EntitlementRef{
					{ParentID: otherApplicationID},
				},
			},
			visible: false,
		},
		{
			name:     "customer has no entitlement while the vendor gates applications",
			affected: []advisory.VersionRef{appVersion(versionOneID)},
			view:     advisory.CustomerView{OrgHasApplicationEntitlements: true},
			visible:  false,
		},
		{
			name:     "one of several affected versions is entitled",
			affected: []advisory.VersionRef{appVersion(versionOneID), appVersion(versionTwoID)},
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				ApplicationEntitlements: []advisory.EntitlementRef{
					{ParentID: applicationID, VersionIDs: []uuid.UUID{versionTwoID}},
				},
			},
			visible: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(advisory.IsVisibleToCustomer(
				types.AdvisoryStatusPublished, tc.affected, nil, tc.view, now,
			)).To(Equal(tc.visible))
		})
	}
}

func TestIsVisibleToCustomerArtifactEntitlements(t *testing.T) {
	cases := []struct {
		name     string
		affected []advisory.VersionRef
		view     advisory.CustomerView
		visible  bool
	}{
		{
			name:     "whole artifact entitlement",
			affected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			view: advisory.CustomerView{
				OrgHasArtifactEntitlements: true,
				ArtifactEntitlements: []advisory.EntitlementRef{
					{ParentID: artifactID},
				},
			},
			visible: true,
		},
		{
			name:     "entitlement pinned to the affected version",
			affected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			view: advisory.CustomerView{
				OrgHasArtifactEntitlements: true,
				ArtifactEntitlements: []advisory.EntitlementRef{
					{ParentID: artifactID, VersionIDs: []uuid.UUID{amd64VersionID}},
				},
			},
			visible: true,
		},
		{
			name:     "entitlement pinned to a different version",
			affected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			view: advisory.CustomerView{
				OrgHasArtifactEntitlements: true,
				ArtifactEntitlements: []advisory.EntitlementRef{
					{ParentID: artifactID, VersionIDs: []uuid.UUID{arm64VersionID}},
				},
			},
			visible: false,
		},
		{
			name:     "entitlement for a different artifact",
			affected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			view: advisory.CustomerView{
				OrgHasArtifactEntitlements: true,
				ArtifactEntitlements: []advisory.EntitlementRef{
					{ParentID: otherArtifactID},
				},
			},
			visible: false,
		},
		{
			// The loader expands an affected child manifest into every version sharing its
			// content, so an entitlement pinning the index tag matches.
			name: "multi-arch: affected child manifest, entitlement on the index",
			affected: []advisory.VersionRef{
				artifactVersion(amd64VersionID),
				artifactVersion(indexVersionID),
			},
			view: advisory.CustomerView{
				OrgHasArtifactEntitlements: true,
				ArtifactEntitlements: []advisory.EntitlementRef{
					{ParentID: artifactID, VersionIDs: []uuid.UUID{indexVersionID}},
				},
			},
			visible: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(advisory.IsVisibleToCustomer(
				types.AdvisoryStatusPublished, nil, tc.affected, tc.view, now,
			)).To(Equal(tc.visible))
		})
	}
}

// A deployment is the strongest possible evidence that a customer is exposed, and it is
// created without any entitlement. Deciding visibility on entitlements alone therefore hid
// advisories from exactly the customers the vendor's impact report lists as affected.
func TestIsVisibleToCustomerDeployments(t *testing.T) {
	gatedVendor := func(deployed ...uuid.UUID) advisory.CustomerView {
		return advisory.CustomerView{
			OrgHasApplicationEntitlements: true,
			OrgHasArtifactEntitlements:    true,
			DeployedApplicationVersionIDs: deployed,
		}
	}

	cases := []struct {
		name                     string
		status                   types.AdvisoryStatus
		appAffected, artAffected []advisory.VersionRef
		view                     advisory.CustomerView
		visible                  bool
	}{
		{
			name:        "deployed the affected version without holding an entitlement",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			view:        gatedVendor(versionOneID),
			visible:     true,
		},
		{
			// The customer already upgraded, which is the "fixed" row of the vendor's
			// impact report. They still need to read the advisory.
			name:        "deployed an affected version they have since upgraded away from",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			view:        gatedVendor(versionOneID, versionTwoID),
			visible:     true,
		},
		{
			name:        "deployed only unaffected versions",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			view:        gatedVendor(versionTwoID),
			visible:     false,
		},
		{
			name:        "no deployments at all",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			view:        gatedVendor(),
			visible:     false,
		},
		{
			// Deployments say nothing about artifact access, which stays entitlement gated.
			name:        "deployments do not unlock an artifact only advisory",
			artAffected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			view:        gatedVendor(versionOneID),
			visible:     false,
		},
		{
			name:        "draft stays hidden even from a customer running the affected version",
			status:      types.AdvisoryStatusDraft,
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			view:        gatedVendor(versionOneID),
			visible:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			status := tc.status
			if status == "" {
				status = types.AdvisoryStatusPublished
			}
			g.Expect(advisory.IsVisibleToCustomer(
				status, tc.appAffected, tc.artAffected, tc.view, now,
			)).To(Equal(tc.visible))
		})
	}
}

func TestIsVisibleToCustomerEntitlementExpiry(t *testing.T) {
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name      string
		expiresAt *time.Time
		visible   bool
	}{
		{"nil never expires", nil, true},
		{"expires in the future", &future, true},
		{"expired in the past", &past, false},
		{"expires exactly now", &now, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			view := advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				ApplicationEntitlements: []advisory.EntitlementRef{
					{ParentID: applicationID, ExpiresAt: tc.expiresAt},
				},
			}
			g.Expect(advisory.IsVisibleToCustomer(
				types.AdvisoryStatusPublished,
				[]advisory.VersionRef{appVersion(versionOneID)},
				nil,
				view,
				now,
			)).To(Equal(tc.visible))
		})
	}
}

func TestIsVisibleToCustomerEntitlementFallback(t *testing.T) {
	appAffected := []advisory.VersionRef{appVersion(versionOneID)}
	artifactAffected := []advisory.VersionRef{artifactVersion(amd64VersionID)}

	cases := []struct {
		name                     string
		appAffected, artAffected []advisory.VersionRef
		view                     advisory.CustomerView
		visible                  bool
	}{
		{
			name:        "vendor uses no entitlements at all",
			appAffected: appAffected,
			view:        advisory.CustomerView{},
			visible:     true,
		},
		{
			// The fallback is per kind. A vendor gating artifacts but not applications must
			// not have its application advisories hidden, but equally must not have them
			// exposed on the basis of the artifact side.
			name:        "vendor gates artifacts only, application advisory falls back to visible",
			appAffected: appAffected,
			artAffected: nil,
			view:        advisory.CustomerView{OrgHasArtifactEntitlements: true},
			visible:     true,
		},
		{
			// The regression this rule exists for: an artifact-only advisory must stay
			// gated even though the vendor configured no application entitlements.
			name:        "vendor gates artifacts only, artifact advisory stays gated",
			artAffected: artifactAffected,
			view:        advisory.CustomerView{OrgHasArtifactEntitlements: true},
			visible:     false,
		},
		{
			name:        "vendor gates applications only, artifact advisory falls back to visible",
			artAffected: artifactAffected,
			view:        advisory.CustomerView{OrgHasApplicationEntitlements: true},
			visible:     true,
		},
		{
			name:        "vendor gates applications only, application advisory stays gated",
			appAffected: appAffected,
			view:        advisory.CustomerView{OrgHasApplicationEntitlements: true},
			visible:     false,
		},
		{
			name:        "vendor gates both kinds and the customer holds neither entitlement",
			appAffected: appAffected,
			artAffected: artifactAffected,
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				OrgHasArtifactEntitlements:    true,
			},
			visible: false,
		},
		{
			name:        "affects both kinds, customer entitled to the artifact only",
			appAffected: appAffected,
			artAffected: artifactAffected,
			view: advisory.CustomerView{
				OrgHasApplicationEntitlements: true,
				OrgHasArtifactEntitlements:    true,
				ArtifactEntitlements: []advisory.EntitlementRef{
					{ParentID: artifactID},
				},
			},
			visible: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(advisory.IsVisibleToCustomer(
				types.AdvisoryStatusPublished, tc.appAffected, tc.artAffected, tc.view, now,
			)).To(Equal(tc.visible))
		})
	}
}

func TestVersionVisibilityAllows(t *testing.T) {
	expired := now.Add(-time.Hour)
	valid := now.Add(time.Hour)

	cases := []struct {
		name       string
		visibility advisory.VersionVisibility
		ref        advisory.VersionRef
		allowed    bool
	}{
		{
			// The counterpart to the fallback in IsVisibleToCustomer: a vendor who does not
			// gate this kind at all discloses every version of it.
			name:       "vendor uses no entitlements of this kind",
			visibility: advisory.VersionVisibility{},
			ref:        appVersion(versionOneID),
			allowed:    true,
		},
		{
			name: "entitlement pins this version",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				Entitlements: []advisory.EntitlementRef{
					{ParentID: applicationID, VersionIDs: []uuid.UUID{versionOneID}},
				},
			},
			ref:     appVersion(versionOneID),
			allowed: true,
		},
		{
			// The case from the review: entitled to one version line, so the other one and
			// its fix must not be disclosed.
			name: "entitlement pins a different version of the same application",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				Entitlements: []advisory.EntitlementRef{
					{ParentID: applicationID, VersionIDs: []uuid.UUID{versionOneID}},
				},
			},
			ref:     appVersion(versionTwoID),
			allowed: false,
		},
		{
			name: "entitlement covers the whole application",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				Entitlements:       []advisory.EntitlementRef{{ParentID: applicationID}},
			},
			ref:     appVersion(versionTwoID),
			allowed: true,
		},
		{
			name: "entitlement is for another application entirely",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				Entitlements:       []advisory.EntitlementRef{{ParentID: otherApplicationID}},
			},
			ref:     appVersion(versionOneID),
			allowed: false,
		},
		{
			name: "entitlement has expired",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				Entitlements:       []advisory.EntitlementRef{{ParentID: applicationID, ExpiresAt: &expired}},
			},
			ref:     appVersion(versionOneID),
			allowed: false,
		},
		{
			name: "entitlement has not expired yet",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				Entitlements:       []advisory.EntitlementRef{{ParentID: applicationID, ExpiresAt: &valid}},
			},
			ref:     appVersion(versionOneID),
			allowed: true,
		},
		{
			// Deployment grants visibility without an entitlement, so the version that
			// granted it has to be disclosed or the advisory would list nothing as affected.
			name: "customer deployed it without holding an entitlement",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				KnownVersionIDs:    []uuid.UUID{versionOneID},
			},
			ref:     appVersion(versionOneID),
			allowed: true,
		},
		{
			name: "customer has neither an entitlement nor a deployment",
			visibility: advisory.VersionVisibility{
				OrgHasEntitlements: true,
				KnownVersionIDs:    []uuid.UUID{versionOneID},
			},
			ref:     appVersion(versionTwoID),
			allowed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(tc.visibility.Allows(tc.ref, now)).To(Equal(tc.allowed))
		})
	}
}

func TestFilterVisibleVersions(t *testing.T) {
	g := NewWithT(t)

	rows := []advisory.VersionRef{
		appVersion(versionOneID),
		appVersion(versionTwoID),
		{VersionID: amd64VersionID, ParentID: otherApplicationID},
	}
	visibility := advisory.VersionVisibility{
		OrgHasEntitlements: true,
		Entitlements: []advisory.EntitlementRef{
			{ParentID: applicationID, VersionIDs: []uuid.UUID{versionOneID}},
		},
	}

	visible := advisory.FilterVisibleVersions(
		rows, func(ref advisory.VersionRef) advisory.VersionRef { return ref }, visibility, now)

	g.Expect(visible).To(Equal([]advisory.VersionRef{appVersion(versionOneID)}))
}

func TestIsStillAffectedDeployments(t *testing.T) {
	cases := []struct {
		name        string
		appAffected []advisory.VersionRef
		current     []uuid.UUID
		affected    bool
	}{
		{
			name:        "runs an affected version",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			current:     []uuid.UUID{versionOneID},
			affected:    true,
		},
		{
			// The counterpart to TestIsVisibleToCustomerDeployments: having once run the
			// affected version keeps the advisory visible, but no longer running it is what
			// tells the customer they are in the clear.
			name:        "has upgraded away from the affected version",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			current:     []uuid.UUID{versionTwoID},
			affected:    false,
		},
		{
			name:        "one of several deployments still runs an affected version",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			current:     []uuid.UUID{versionTwoID, versionOneID},
			affected:    true,
		},
		{
			name:        "runs one of several affected versions",
			appAffected: []advisory.VersionRef{appVersion(versionOneID), appVersion(versionTwoID)},
			current:     []uuid.UUID{versionTwoID},
			affected:    true,
		},
		{
			name:        "deploys nothing at all",
			appAffected: []advisory.VersionRef{appVersion(versionOneID)},
			current:     nil,
			affected:    false,
		},
		{
			// An entitled customer who never deployed still sees the advisory, and must be
			// told they are not affected rather than left guessing.
			name:        "advisory affects no application version",
			appAffected: nil,
			current:     []uuid.UUID{versionOneID},
			affected:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			marked := advisory.MarkedVersions{AffectedApplicationVersions: tc.appAffected}
			exposure := advisory.Exposure{CurrentApplicationVersionIDs: tc.current}
			g.Expect(advisory.IsStillAffected(marked, exposure)).To(Equal(tc.affected))
		})
	}
}

func otherArtifactVersion(versionID uuid.UUID) advisory.VersionRef {
	return advisory.VersionRef{VersionID: versionID, ParentID: otherArtifactID}
}

func TestIsStillAffectedPulls(t *testing.T) {
	otherAffectedID := uuid.New()
	otherFixedID := uuid.New()

	cases := []struct {
		name        string
		artAffected []advisory.VersionRef
		artFixed    []advisory.VersionRef
		pulled      []uuid.UUID
		affected    bool
	}{
		{
			name:        "pulled an affected version and no fix exists yet",
			artAffected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			pulled:      []uuid.UUID{amd64VersionID},
			affected:    true,
		},
		{
			name:        "pulled an affected version and has not pulled the fix",
			artAffected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			artFixed:    []advisory.VersionRef{artifactVersion(arm64VersionID)},
			pulled:      []uuid.UUID{amd64VersionID},
			affected:    true,
		},
		{
			name:        "pulled the fix as well",
			artAffected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			artFixed:    []advisory.VersionRef{artifactVersion(arm64VersionID)},
			pulled:      []uuid.UUID{amd64VersionID, arm64VersionID},
			affected:    false,
		},
		{
			// Pulling the fix by a sibling tag counts, which is why GetMarkedVersions expands
			// the fixed side the same way it expands the affected side.
			name:        "pulled the fix under a different tag of the same digest",
			artAffected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			artFixed:    []advisory.VersionRef{artifactVersion(arm64VersionID), artifactVersion(indexVersionID)},
			pulled:      []uuid.UUID{amd64VersionID, indexVersionID},
			affected:    false,
		},
		{
			name:        "never pulled anything affected",
			artAffected: []advisory.VersionRef{artifactVersion(amd64VersionID)},
			pulled:      []uuid.UUID{arm64VersionID},
			affected:    false,
		},
		{
			// The fix is tracked per artifact: taking it for one does not settle the other.
			name: "pulled the fix for one artifact but not the other",
			artAffected: []advisory.VersionRef{
				artifactVersion(amd64VersionID),
				otherArtifactVersion(otherAffectedID),
			},
			artFixed: []advisory.VersionRef{artifactVersion(arm64VersionID), otherArtifactVersion(otherFixedID)},
			pulled:   []uuid.UUID{amd64VersionID, arm64VersionID, otherAffectedID},
			affected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			marked := advisory.MarkedVersions{
				AffectedArtifactVersions: tc.artAffected,
				FixedArtifactVersions:    tc.artFixed,
			}
			exposure := advisory.Exposure{PulledArtifactVersionIDs: tc.pulled}
			g.Expect(advisory.IsStillAffected(marked, exposure)).To(Equal(tc.affected))
		})
	}
}
