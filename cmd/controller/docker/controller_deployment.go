package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
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
	CurrentRevisionID uuid.UUID        `json:"currentRevisionId,omitzero"`
	ProjectName       string           `json:"projectName"`
	DockerType        types.DockerType `json:"docker_type,omitempty"`
	State             State            `json:"phase"`
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

func (d *ControllerDeployment) FileName() string {
	return path.Join(controllerDeploymentDir(), d.ID.String())
}

func controllerDeploymentDir() string {
	return path.Join(ScratchDir(), "deployments")
}

func NewControllerDeployment(
	deployment api.ControllerDeployment,
	previous *ControllerDeployment,
) (*ControllerDeployment, error) {
	if name, err := getProjectName(deployment.ComposeFile); err != nil {
		return nil, err
	} else {
		result := &ControllerDeployment{
			ID:          deployment.ID,
			RevisionID:  deployment.RevisionID,
			ProjectName: name,
			DockerType:  *deployment.DockerType,
		}
		if previous != nil && previous.ID == deployment.ID {
			result.CurrentRevisionID = previous.AppliedRevisionID()
		}
		return result, nil
	}
}

func getProjectName(data []byte) (string, error) {
	if compose, err := DecodeComposeFile(data); err != nil {
		return "", err
	} else if name, ok := compose["name"].(string); !ok {
		return "", fmt.Errorf("name is not a string")
	} else {
		return name, nil
	}
}

var controllerDeploymentMutex = sync.RWMutex{}

func GetExistingDeployments() (map[uuid.UUID]ControllerDeployment, error) {
	controllerDeploymentMutex.RLock()
	defer controllerDeploymentMutex.RUnlock()

	if entries, err := os.ReadDir(controllerDeploymentDir()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	} else {
		fn := func(name string) (*ControllerDeployment, error) {
			if file, err := os.Open(path.Join(controllerDeploymentDir(), name)); err != nil {
				return nil, err
			} else {
				defer file.Close()
				var d ControllerDeployment
				if err := json.NewDecoder(file).Decode(&d); err != nil {
					return nil, err
				}
				return &d, nil
			}
		}
		result := make(map[uuid.UUID]ControllerDeployment, len(entries))
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

func SaveDeployment(deployment ControllerDeployment) error {
	controllerDeploymentMutex.Lock()
	defer controllerDeploymentMutex.Unlock()

	if err := os.MkdirAll(path.Dir(deployment.FileName()), 0o700); err != nil {
		return err
	}

	file, err := os.Create(deployment.FileName())
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(deployment); err != nil {
		return err
	}

	return nil
}

func DeleteDeployment(deployment ControllerDeployment) error {
	return os.Remove(deployment.FileName())
}
