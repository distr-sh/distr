package authz

import (
	"testing"

	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestAuthorizeOrganizationFallsBackToPublicAccessForOtherOrganizations(t *testing.T) {
	g := NewWithT(t)
	org := &types.Organization{Slug: new("vendor")}
	public := func() error { return nil }
	private := func() error { return ErrAuthenticationRequired }

	ownRef, err := name.Parse("vendor/app")
	g.Expect(err).NotTo(HaveOccurred())
	owned, err := authorizeOrganization(org, ownRef, func() error {
		t.Fatal("an owned reference must not be authorized as public, which would skip the entitlement check")
		return nil
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(owned).To(BeTrue())

	foreignRef, err := name.Parse("distr-sh/distr/docker-controller")
	g.Expect(err).NotTo(HaveOccurred())
	owned, err = authorizeOrganization(org, foreignRef, public)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(owned).To(BeFalse())

	_, err = authorizeOrganization(org, foreignRef, private)
	g.Expect(err).To(MatchError(ErrAccessDenied))

	_, err = authorizeOrganization(&types.Organization{}, foreignRef, private)
	g.Expect(err).To(MatchError(ErrAccessDenied))
}
