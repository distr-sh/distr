package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// The label, the secret names and the field managers below still name the agent, which is what the controller
// was called before. They identify state that previous versions stored in the cluster: a controller using new
// names would no longer find the Helm releases it installed.
const (
	LabelDeployment  = "agent.distr.sh/deployment"
	secretNamePrefix = "sh.distr.agent.v1."
	fieldManager     = "distr-agent"
	fieldManagerLogs = "distr-agent-logs"
)

type State string

const (
	StateUnspecified State = ""
	StateProgressing State = "progressing"
	StateReady       State = "ready"
	StateFailed      State = "failed"
)

type ControllerDeployment struct {
	ID         uuid.UUID `json:"id"`
	RevisionID uuid.UUID `json:"revisionId"`
	// CurrentRevisionID is the revision that was last applied successfully. It differs from RevisionID
	// while a newer revision is being applied or after applying it has failed.
	CurrentRevisionID uuid.UUID `json:"currentRevisionId,omitzero"`
	ReleaseName       string    `json:"releaseName"`
	HelmRevision      *int      `json:"helmRevision,omitempty"`
	State             State     `json:"phase"`
}

func (d ControllerDeployment) GetDeploymentID() uuid.UUID {
	return d.ID
}

func (d ControllerDeployment) GetDeploymentRevisionID() uuid.UUID {
	return d.RevisionID
}

// AppliedRevisionID returns the revision that was last applied successfully or [uuid.Nil] if there is none.
// State saved by controllers that did not know CurrentRevisionID yet only has a RevisionID, which is the applied
// one unless applying it failed or is still in progress.
func (d ControllerDeployment) AppliedRevisionID() uuid.UUID {
	if d.CurrentRevisionID != uuid.Nil {
		return d.CurrentRevisionID
	}
	if d.State == StateReady || d.State == StateUnspecified {
		return d.RevisionID
	}
	return uuid.Nil
}

func (d *ControllerDeployment) SecretName() string {
	return secretNamePrefix + d.ReleaseName
}

func NewControllerDeployment(
	deployment api.ControllerDeployment,
	previous *ControllerDeployment,
) ControllerDeployment {
	result := ControllerDeployment{
		ReleaseName: deployment.ReleaseName,
		ID:          deployment.ID,
		RevisionID:  deployment.RevisionID,
	}
	if previous != nil {
		result.CurrentRevisionID = previous.AppliedRevisionID()
		result.HelmRevision = previous.HelmRevision
	}
	return result
}

func PullSecretName(releaseName string) string {
	return secretNamePrefix + releaseName + ".pull"
}

func GetExistingDeployments(ctx context.Context, namespace string) (map[uuid.UUID]ControllerDeployment, error) {
	if secrets, err := k8sClient.CoreV1().Secrets(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: LabelDeployment}); err != nil {
		return nil, err
	} else {
		deployments := make(map[uuid.UUID]ControllerDeployment, len(secrets.Items))
		for _, secret := range secrets.Items {
			var deployment ControllerDeployment
			if err := json.Unmarshal(secret.Data["release"], &deployment); err != nil {
				return nil, err
			} else {
				deployments[deployment.ID] = deployment
			}
		}
		return deployments, nil
	}
}

func SaveDeployment(ctx context.Context, namespace string, deployment ControllerDeployment) error {
	cfg := applyconfigurationscorev1.Secret(deployment.SecretName(), namespace)
	cfg.WithLabels(map[string]string{LabelDeployment: deployment.ReleaseName})
	if data, err := json.Marshal(deployment); err != nil {
		return err
	} else {
		cfg.WithData(map[string][]byte{"release": data})
	}
	_, err := k8sClient.CoreV1().Secrets(namespace).Apply(
		ctx,
		cfg,
		metav1.ApplyOptions{Force: true, FieldManager: fieldManager},
	)
	return err
}

func DeleteDeployment(ctx context.Context, namespace string, deployment ControllerDeployment) error {
	err := k8sClient.CoreV1().Secrets(namespace).Delete(ctx, deployment.SecretName(), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete ControllerDeployment: %w", err)
	}
	return nil
}
