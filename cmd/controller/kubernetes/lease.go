package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/distr-sh/distr/internal/controllerenv"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// acquireLease blocks until this controller holds the lease of its deployment target. Kubernetes starts the
// replacement of a terminating pod right away, which would otherwise apply Helm releases and write their tracking
// secrets while the terminating controller is still finishing its operation.
//
// The returned context is cancelled with ctx or when the lease is lost. The lease outlives ctx until release is
// called, so that a shutdown signal does not hand it over while a Helm operation is still running.
func acquireLease(ctx context.Context) (context.Context, func(), error) {
	namespace, _, err := k8sConfigFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, nil, fmt.Errorf("could not determine controller namespace: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("could not determine hostname: %w", err)
	}

	name := "distr-" + controllerenv.ControllerName()
	leaseCtx, cancelLease := context.WithCancel(context.WithoutCancel(ctx))
	leaderCtx, cancelLeader := context.WithCancel(ctx)
	leading := make(chan struct{})
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Name: name,
		Lock: &resourcelock.LeaseLock{
			LeaseMeta:  metav1.ObjectMeta{Name: name, Namespace: namespace},
			Client:     k8sClient.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{Identity: hostname + "_" + uuid.NewString()},
		},
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(context.Context) { close(leading) },
			OnStoppedLeading: func() {
				if leaseCtx.Err() == nil {
					logger.Warn("lost controller lease")
				}
				cancelLeader()
			},
		},
	})
	if err != nil {
		cancelLease()
		cancelLeader()
		return nil, nil, fmt.Errorf("could not create leader elector: %w", err)
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		elector.Run(leaseCtx)
	}()
	release := func() {
		cancelLease()
		<-stopped
	}

	select {
	case <-leading:
		return leaderCtx, release, nil
	case <-ctx.Done():
		release()
		return nil, nil, ctx.Err()
	}
}
