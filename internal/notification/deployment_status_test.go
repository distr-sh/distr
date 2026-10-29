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

func TestDeploymentStatusNotificationFor_StaleRecoveryAfterControllerDiedMidApply(t *testing.T) {
	g := NewWithT(t)
	oldRevision, newRevision := uuid.New(), uuid.New()
	previous := status(newRevision, types.DeploymentStatusTypeProgressing, 10*time.Minute)

	for _, settled := range []*types.DeploymentRevisionStatus{
		status(oldRevision, types.DeploymentStatusTypeHealthy, time.Hour),
		nil,
	} {
		current := *status(newRevision, types.DeploymentStatusTypeHealthy, 0)

		kind, ok := deploymentStatusNotificationFor(previous, settled, current, &oldRevision, false, true)
		g.Expect(ok).To(BeTrue())
		g.Expect(kind).To(Equal(deploymentStatusNotificationStaleRecovered))

		_, ok = deploymentStatusNotificationFor(previous, settled, current, &oldRevision, false, false)
		g.Expect(ok).To(BeFalse())
	}
}

func TestDeploymentStatusNotificationFor_ErrorAfterStaleWarningIsNoRecovery(t *testing.T) {
	g := NewWithT(t)
	revision := uuid.New()
	settled := status(revision, types.DeploymentStatusTypeError, time.Hour)
	current := *status(revision, types.DeploymentStatusTypeError, 0)

	kind, ok := deploymentStatusNotificationFor(
		status(revision, types.DeploymentStatusTypeError, time.Hour), settled, current, &revision, false, true,
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(kind).To(Equal(deploymentStatusNotificationError))

	_, ok = deploymentStatusNotificationFor(
		status(revision, types.DeploymentStatusTypeError, 5*time.Second), settled, current, &revision, false, true,
	)
	g.Expect(ok).To(BeFalse())
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
		_, ok := deploymentStatusNotificationFor(previous, settled, *status(revision, current, 0), &revision, false, false)
		g.Expect(ok).To(BeFalse())
	}
}

func TestDeploymentStatusNotificationFor_NoAlertOnRetriedError(t *testing.T) {
	g := NewWithT(t)
	revision := uuid.New()
	previous := status(revision, types.DeploymentStatusTypeProgressing, 5*time.Second)
	settled := status(revision, types.DeploymentStatusTypeError, 10*time.Minute)

	_, ok := deploymentStatusNotificationFor(
		previous, settled, *status(revision, types.DeploymentStatusTypeError, 0), &revision, false, false,
	)
	g.Expect(ok).To(BeFalse())
}

func TestDeploymentStatusNotificationFor_ErrorAndRecoveryJudgedBySettledStatus(t *testing.T) {
	g := NewWithT(t)
	revision := uuid.New()
	previous := status(revision, types.DeploymentStatusTypeProgressing, 5*time.Second)

	kind, ok := deploymentStatusNotificationFor(
		previous,
		status(revision, types.DeploymentStatusTypeHealthy, 10*time.Minute),
		*status(revision, types.DeploymentStatusTypeError, 0),
		&revision,
		false,
		false,
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(kind).To(Equal(deploymentStatusNotificationError))

	kind, ok = deploymentStatusNotificationFor(
		previous,
		status(revision, types.DeploymentStatusTypeError, 10*time.Minute),
		*status(revision, types.DeploymentStatusTypeHealthy, 0),
		&revision,
		false,
		false,
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(kind).To(Equal(deploymentStatusNotificationErrorRecovered))
}

func TestDeploymentStatusNotificationFor_ErrorKindDependsOnCurrentRevision(t *testing.T) {
	g := NewWithT(t)
	oldRevision, newRevision := uuid.New(), uuid.New()
	previous := status(newRevision, types.DeploymentStatusTypeProgressing, 5*time.Second)
	current := *status(newRevision, types.DeploymentStatusTypeError, 0)

	for _, tc := range []struct {
		currentRevisionID *uuid.UUID
		expected          deploymentStatusNotificationKind
	}{
		{nil, deploymentStatusNotificationError},
		{&oldRevision, deploymentStatusNotificationUpdateFailed},
		{&newRevision, deploymentStatusNotificationError},
	} {
		kind, ok := deploymentStatusNotificationFor(previous, nil, current, tc.currentRevisionID, false, false)
		g.Expect(ok).To(BeTrue())
		g.Expect(kind).To(Equal(tc.expected))
	}
}
