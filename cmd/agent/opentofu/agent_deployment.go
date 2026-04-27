package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
)

type State string

const (
	StateUnspecified State = ""
	StateInstalling  State = "installing"
	StateInstalled   State = "installed"
	StateFailed      State = "failed"
)

type AgentDeployment struct {
	ID                uuid.UUID         `json:"id"`
	RevisionID        uuid.UUID         `json:"revisionId"`
	TofuConfigURL     string            `json:"tofuConfigUrl"`
	TofuConfigVersion string            `json:"tofuConfigVersion"`
	TofuBackendConfig map[string]string `json:"tofuBackendConfig,omitempty"`
	State             State             `json:"phase"`
}

func (d AgentDeployment) GetDeploymentID() uuid.UUID {
	return d.ID
}

func (d AgentDeployment) GetDeploymentRevisionID() uuid.UUID {
	return d.RevisionID
}

func (d *AgentDeployment) FileName() string {
	return filepath.Join(DeploymentsDir(), d.ID.String())
}

func NewAgentDeployment(deployment api.AgentDeployment) *AgentDeployment {
	return &AgentDeployment{
		ID:                deployment.ID,
		RevisionID:        deployment.RevisionID,
		TofuConfigURL:     deployment.TofuConfigURL,
		TofuConfigVersion: deployment.TofuConfigVersion,
		TofuBackendConfig: deployment.TofuBackendConfig,
	}
}

var agentDeploymentMutex = sync.RWMutex{}

func GetExistingDeployments() (map[uuid.UUID]AgentDeployment, error) {
	agentDeploymentMutex.RLock()
	defer agentDeploymentMutex.RUnlock()

	if entries, err := os.ReadDir(DeploymentsDir()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	} else {
		fn := func(name string) (*AgentDeployment, error) {
			if file, err := os.Open(filepath.Join(DeploymentsDir(), name)); err != nil {
				return nil, err
			} else {
				defer file.Close()
				var d AgentDeployment
				if err := json.NewDecoder(file).Decode(&d); err != nil {
					return nil, err
				}
				return &d, nil
			}
		}
		result := make(map[uuid.UUID]AgentDeployment, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				if d, err := fn(entry.Name()); err != nil {
					return nil, err
				} else {
					result[d.ID] = *d
				}
			}
		}
		return result, nil
	}
}

func SaveDeployment(deployment AgentDeployment) error {
	agentDeploymentMutex.Lock()
	defer agentDeploymentMutex.Unlock()

	if err := os.MkdirAll(filepath.Dir(deployment.FileName()), 0o700); err != nil {
		return err
	}

	file, err := os.OpenFile(deployment.FileName(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(deployment); err != nil {
		return err
	}

	return nil
}

func DeleteDeployment(deployment AgentDeployment) error {
	agentDeploymentMutex.Lock()
	defer agentDeploymentMutex.Unlock()
	return os.Remove(deployment.FileName())
}
