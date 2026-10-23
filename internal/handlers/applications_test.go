package handlers

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestEntitlementsToAutoAddVersionTo(t *testing.T) {
	majorTwo := uuid.New()
	hotfix := uuid.New()
	entitlements := []types.ApplicationEntitlementBase{
		{ID: majorTwo, AutoAddVersionRegex: new(`^2\.`)},
		{ID: hotfix, AutoAddVersionRegex: new(`-hotfix$`)},
	}

	cases := []struct {
		name         string
		versionName  string
		previousName *string
		expected     []uuid.UUID
	}{
		{"new version matching one regex", "2.1.0", nil, []uuid.UUID{majorTwo}},
		{"new version matching both regexes", "2.1.0-hotfix", nil, []uuid.UUID{majorTwo, hotfix}},
		{"new version matching no regex", "3.0.0", nil, nil},
		{"rename into a match", "2.1.0", new("release-2.1.0"), []uuid.UUID{majorTwo}},
		{"rename keeping a match leaves a manual removal alone", "2.1.0-hotfix", new("2.1.0"), []uuid.UUID{hotfix}},
		{"rename out of a match", "3.0.0", new("2.1.0"), nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(entitlementsToAutoAddVersionTo(t.Context(), entitlements, tc.versionName, tc.previousName)).
				To(Equal(tc.expected))
		})
	}
}
