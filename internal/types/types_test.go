package types

import (
	"encoding/json"
	"testing"

	"github.com/distr-sh/distr/internal/util"
	. "github.com/onsi/gomega"
)

func TestDeploymentStatusTypeParsing(t *testing.T) {
	g := NewWithT(t)

	var target struct {
		Type DeploymentStatusType `json:"type"`
	}

	err := json.Unmarshal([]byte(`{"type": "healthy"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentStatusTypeHealthy))

	err = json.Unmarshal([]byte(`{"type": "ok"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentStatusTypeRunning))

	err = json.Unmarshal([]byte(`{"type": "does-not-exist"}`), &target)
	g.Expect(err).To(MatchError(ErrInvalidDeploymentStatusType))
}

func TestUserRoleParsing(t *testing.T) {
	g := NewWithT(t)

	var target struct {
		Role UserRole `json:"role"`
	}

	err := json.Unmarshal([]byte(`{"role": "read_only"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Role).To(Equal(UserRoleReadOnly))

	err = json.Unmarshal([]byte(`{"role": "read_write"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Role).To(Equal(UserRoleReadWrite))

	err = json.Unmarshal([]byte(`{"role": "admin"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Role).To(Equal(UserRoleAdmin))

	err = json.Unmarshal([]byte(`{"role": "superuser"}`), &target)
	g.Expect(err).To(HaveOccurred())
}

func TestLessOrEqualUserRole(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		a, b     UserRole
		expected bool
	}{
		{UserRoleReadOnly, UserRoleReadOnly, true},
		{UserRoleReadOnly, UserRoleReadWrite, true},
		{UserRoleReadOnly, UserRoleAdmin, true},
		{UserRoleReadWrite, UserRoleReadOnly, false},
		{UserRoleReadWrite, UserRoleReadWrite, true},
		{UserRoleReadWrite, UserRoleAdmin, true},
		{UserRoleAdmin, UserRoleReadOnly, false},
		{UserRoleAdmin, UserRoleReadWrite, false},
		{UserRoleAdmin, UserRoleAdmin, true},
	}
	for _, tc := range cases {
		g.Expect(LessOrEqualUserRole(tc.a, tc.b)).
			To(Equal(tc.expected), "LessOrEqualUserRole(%q, %q)", tc.a, tc.b)
	}

	// Unknown roles rank below every valid role, so they are always ≤ a valid role
	// and a valid role is never ≤ an unknown role.
	g.Expect(LessOrEqualUserRole(UserRole("unknown"), UserRoleReadOnly)).To(BeTrue())
	g.Expect(LessOrEqualUserRole(UserRoleReadOnly, UserRole("unknown"))).To(BeFalse())
}

func TestMinUserRole(t *testing.T) {
	g := NewWithT(t)

	admin := util.PtrTo(UserRoleAdmin)
	rw := util.PtrTo(UserRoleReadWrite)
	ro := util.PtrTo(UserRoleReadOnly)

	g.Expect(MinUserRole(admin, ro)).To(Equal(UserRoleReadOnly))
	g.Expect(MinUserRole(ro, admin)).To(Equal(UserRoleReadOnly))
	g.Expect(MinUserRole(rw, admin)).To(Equal(UserRoleReadWrite))
	g.Expect(MinUserRole(admin, admin)).To(Equal(UserRoleAdmin))

	// nil acts as "absent" — the other side wins.
	g.Expect(MinUserRole(nil, admin)).To(Equal(UserRoleAdmin))
	g.Expect(MinUserRole(rw, nil)).To(Equal(UserRoleReadWrite))

	// Both nil yields the empty string, which downstream authorization treats
	// as no role (fails closed against RequireAnyUserRole).
	g.Expect(MinUserRole(nil, nil)).To(BeEmpty())
}

func TestDeploymentTypeParsing(t *testing.T) {
	g := NewWithT(t)

	var target struct {
		Type DeploymentType `json:"type"`
	}

	err := json.Unmarshal([]byte(`{"type": "docker"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentTypeDocker))

	err = json.Unmarshal([]byte(`{"type": "kubernetes"}`), &target)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target.Type).To(Equal(DeploymentTypeKubernetes))

	err = json.Unmarshal([]byte(`{"type": "swarm"}`), &target)
	g.Expect(err).To(MatchError(ErrInvalidDeploymentType))
}
