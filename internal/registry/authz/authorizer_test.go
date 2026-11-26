package authz

import (
	"testing"

	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

type testPrincipal struct {
	authinfo.AuthInfoWithOrganization
	org                *types.Organization
	customerOrgID      *uuid.UUID
	deploymentTargetID *uuid.UUID
}

func (p testPrincipal) CurrentOrg() *types.Organization       { return p.org }
func (p testPrincipal) CurrentCustomerOrgID() *uuid.UUID      { return p.customerOrgID }
func (p testPrincipal) CurrentDeploymentTargetID() *uuid.UUID { return p.deploymentTargetID }

func TestShouldIgnoreEntitledVersionsOnlyForControllerPulls(t *testing.T) {
	g := NewWithT(t)
	org := &types.Organization{ControllerAnyArtifactVersionEnabled: true}
	user := testPrincipal{org: org, customerOrgID: new(uuid.New())}
	controller := testPrincipal{org: org, customerOrgID: new(uuid.New()), deploymentTargetID: new(uuid.New())}

	g.Expect(shouldIgnoreEntitledVersions(controller, ActionRead)).To(BeTrue())
	g.Expect(shouldIgnoreEntitledVersions(controller, ActionStat)).To(BeTrue())
	g.Expect(shouldIgnoreEntitledVersions(controller, ActionDiscover)).To(BeFalse())
	g.Expect(shouldIgnoreEntitledVersions(user, ActionRead)).To(BeFalse())

	org.ControllerAnyArtifactVersionEnabled = false
	g.Expect(shouldIgnoreEntitledVersions(controller, ActionRead)).To(BeFalse())
}

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
