package security_test

import (
	"testing"

	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestHashPassword(t *testing.T) {
	g := NewWithT(t)
	u1 := types.UserAccount{Password: "12345678"}
	u2 := types.UserAccount{Password: "12345678"}
	g.Expect(security.HashPassword(&u1)).NotTo(HaveOccurred())
	g.Expect(u1.Password).To(BeEmpty())
	g.Expect(u1.PasswordSalt).NotTo(BeEmpty())
	g.Expect(security.HashPassword(&u2)).NotTo(HaveOccurred())
	g.Expect(u2.Password).To(BeEmpty())
	g.Expect(u2.PasswordSalt).NotTo(BeEmpty())
	g.Expect(u1.PasswordSalt).NotTo(Equal(u2.PasswordSalt))
	g.Expect(u1.PasswordHash).NotTo(Equal(u2.PasswordHash))
}

func TestVerifyDeploymentTargetAccessKey(t *testing.T) {
	g := NewWithT(t)
	activeSalt, activeHash, err := security.HashAccessKey("active")
	g.Expect(err).NotTo(HaveOccurred())
	pendingSalt, pendingHash, err := security.HashAccessKey("pending")
	g.Expect(err).NotTo(HaveOccurred())

	t.Run("accepts the active secret when no reconnect is pending", func(t *testing.T) {
		g := NewWithT(t)
		dt := types.DeploymentTarget{AccessKeySalt: &activeSalt, AccessKeyHash: &activeHash}
		g.Expect(security.VerifyDeploymentTargetAccessKey(dt, "active")).To(BeFalse())
		_, err := security.VerifyDeploymentTargetAccessKey(dt, "pending")
		g.Expect(err).To(MatchError(security.ErrInvalidAccessKey))
	})

	t.Run("accepts both secrets while a reconnect is pending", func(t *testing.T) {
		g := NewWithT(t)
		dt := types.DeploymentTarget{
			AccessKeySalt: &activeSalt, AccessKeyHash: &activeHash,
			PendingAccessKeySalt: &pendingSalt, PendingAccessKeyHash: &pendingHash,
		}
		g.Expect(security.VerifyDeploymentTargetAccessKey(dt, "active")).To(BeFalse())
		g.Expect(security.VerifyDeploymentTargetAccessKey(dt, "pending")).To(BeTrue())
		_, err := security.VerifyDeploymentTargetAccessKey(dt, "wrong")
		g.Expect(err).To(MatchError(security.ErrInvalidAccessKey))
	})

	t.Run("accepts the pending secret of a target that never connected", func(t *testing.T) {
		g := NewWithT(t)
		dt := types.DeploymentTarget{PendingAccessKeySalt: &pendingSalt, PendingAccessKeyHash: &pendingHash}
		g.Expect(security.VerifyDeploymentTargetAccessKey(dt, "pending")).To(BeTrue())
	})

	t.Run("rejects every secret of a target without secrets", func(t *testing.T) {
		g := NewWithT(t)
		_, err := security.VerifyDeploymentTargetAccessKey(types.DeploymentTarget{}, "")
		g.Expect(err).To(MatchError(security.ErrNoAccessKey))
	})
}

func TestVerifyPassword(t *testing.T) {
	g := NewWithT(t)
	pw := "12345678"
	u := types.UserAccount{Password: pw}
	g.Expect(security.HashPassword(&u)).NotTo(HaveOccurred())
	g.Expect(security.VerifyPassword(u, pw)).NotTo(HaveOccurred())
	g.Expect(security.VerifyPassword(u, "wrong")).To(MatchError(security.ErrInvalidPassword))
}
