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

const LabelDeplyoment = "agent.distr.sh/deployment"

type State string

const (
	StateUnspecified State = ""
	StateProgressing State = "progressing"
	StateReady       State = "ready"
	StateFailed      State = "failed"
)

type AgentDeployment struct {
	ID           uuid.UUID `json:"id"`
	RevisionID   uuid.UUID `json:"revisionId"`
	ReleaseName  string    `json:"releaseName"`
	HelmRevision *int      `json:"helmRevision,omitempty"`
	LogsEnabled  bool      `json:"logsEnabled"`
	State        State     `json:"phase"`
}

func (d AgentDeployment) GetDeploymentID() uuid.UUID {
	return d.ID
}

func (d AgentDeployment) GetDeploymentRevisionID() uuid.UUID {
	return d.RevisionID
}

func (d *AgentDeployment) SecretName() string {
	return fmt.Sprintf("sh.distr.agent.v1.%v", d.ReleaseName)
}

func NewAgentDeployment(deployment api.AgentDeployment) AgentDeployment {
	return AgentDeployment{
		ReleaseName: deployment.ReleaseName,
		ID:          deployment.ID,
		RevisionID:  deployment.RevisionID,
		LogsEnabled: deployment.LogsEnabled,
	}
}

func PullSecretName(releaseName string) string {
	return fmt.Sprintf("sh.distr.agent.v1.%v.pull", releaseName)
}

func GetExistingDeployments(ctx context.Context, namespace string) ([]AgentDeployment, error) {
	if secrets, err := k8sClient.CoreV1().Secrets(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: LabelDeplyoment}); err != nil {
		return nil, err
	} else {
		deployments := make([]AgentDeployment, len(secrets.Items))
		for i, secret := range secrets.Items {
			var deployment AgentDeployment
			if err := json.Unmarshal(secret.Data["release"], &deployment); err != nil {
				return nil, err
			} else {
				deployments[i] = deployment
			}
		}
		return deployments, nil
	}
}

func SaveDeployment(ctx context.Context, namespace string, deployment AgentDeployment) error {
	cfg := applyconfigurationscorev1.Secret(deployment.SecretName(), namespace)
	cfg.WithLabels(map[string]string{LabelDeplyoment: deployment.ReleaseName})
	if data, err := json.Marshal(deployment); err != nil {
		return err
	} else {
		cfg.WithData(map[string][]byte{"release": data})
	}
	_, err := k8sClient.CoreV1().Secrets(namespace).Apply(
		ctx,
		cfg,
		metav1.ApplyOptions{Force: true, FieldManager: "distr-agent"},
	)
	return err
}

func DeleteDeployment(ctx context.Context, namespace string, deployment AgentDeployment) error {
	err := k8sClient.CoreV1().Secrets(namespace).Delete(ctx, deployment.SecretName(), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete AgentDeployment: %w", err)
	}
	return nil
}
