package notification

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func status(
	revisionID uuid.UUID,
	statusType types.DeploymentStatusType,
	age time.Duration,
) *types.DeploymentRevisionStatus {
	return &types.DeploymentRevisionStatus{
		DeploymentRevisionID: revisionID,
		Type:                 statusType,
		CreatedAt:            time.Now().Add(-age),
	}
}

func TestDeploymentStatusNotificationFor_StaleRecoveryAfterAgentDiedMidApply(t *testing.T) {
	g := NewWithT(t)
	oldRevision, newRevision := uuid.New(), uuid.New()
	previous := status(newRevision, types.DeploymentStatusTypeProgressing, 10*time.Minute)

	for _, settled := range []*types.DeploymentRevisionStatus{
		status(oldRevision, types.DeploymentStatusTypeHealthy, time.Hour),
		nil,
	} {
		kind, reference, ok := deploymentStatusNotificationFor(
			previous, settled, *status(newRevision, types.DeploymentStatusTypeHealthy, 0),
		)
		g.Expect(ok).To(BeTrue())
		g.Expect(kind).To(Equal(deploymentStatusNotificationStaleRecovered))
		g.Expect(reference).To(BeIdenticalTo(previous))
	}
}

func TestDeploymentStatusNotificationFor_NoNotificationWhileProgressingAfterOldSettledStatus(t *testing.T) {
	g := NewWithT(t)
	revision := uuid.New()
	previous := status(revision, types.DeploymentStatusTypeProgressing, 5*time.Second)
	settled := status(revision, types.DeploymentStatusTypeHealthy, time.Hour)

	for _, current := range []types.DeploymentStatusType{
		types.DeploymentStatusTypeProgressing,
		types.DeploymentStatusTypeHealthy,
	} {
		_, _, ok := deploymentStatusNotificationFor(previous, settled, *status(revision, current, 0))
		g.Expect(ok).To(BeFalse())
	}
}

func TestDeploymentStatusNotificationFor_NoAlertOnRetriedError(t *testing.T) {
	g := NewWithT(t)
	revision := uuid.New()
	previous := status(revision, types.DeploymentStatusTypeProgressing, 5*time.Second)
	settled := status(revision, types.DeploymentStatusTypeError, 10*time.Minute)

	_, _, ok := deploymentStatusNotificationFor(
		previous, settled, *status(revision, types.DeploymentStatusTypeError, 0),
	)
	g.Expect(ok).To(BeFalse())
}

func TestDeploymentStatusNotificationFor_ErrorAndRecoveryKeyedOnSettledStatus(t *testing.T) {
	g := NewWithT(t)
	revision := uuid.New()
	settledHealthy := status(revision, types.DeploymentStatusTypeHealthy, 10*time.Minute)
	previous := status(revision, types.DeploymentStatusTypeProgressing, 5*time.Second)

	kind, reference, ok := deploymentStatusNotificationFor(
		previous, settledHealthy, *status(revision, types.DeploymentStatusTypeError, 0),
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(kind).To(Equal(deploymentStatusNotificationError))
	g.Expect(reference).To(BeIdenticalTo(settledHealthy))

	settledError := status(revision, types.DeploymentStatusTypeError, 10*time.Minute)
	kind, reference, ok = deploymentStatusNotificationFor(
		previous, settledError, *status(revision, types.DeploymentStatusTypeHealthy, 0),
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(kind).To(Equal(deploymentStatusNotificationErrorRecovered))
	g.Expect(reference).To(BeIdenticalTo(settledError))
}
