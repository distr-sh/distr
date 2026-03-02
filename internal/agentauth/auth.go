package agentauth

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path"

	containerdlog "github.com/containerd/log"
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentenv"
	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/google/uuid"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

var (
	previousAuth     = map[uuid.UUID]map[string]api.AgentRegistryAuth{}
	previousJWT      = map[uuid.UUID]string{}
	credentialStores = map[uuid.UUID]credentials.Store{}
)

func init() {
	_ = containerdlog.SetLevel("warn")
}

func EnsureAuth(
	ctx context.Context,
	jwt string,
	deployment api.AgentDeployment,
) (*auth.Client, error) {
	if err := os.MkdirAll(DockerConfigDir(deployment), 0o700); err != nil {
		return nil, fmt.Errorf("could not create docker config dir for deployment: %w", err)
	}

	var store credentials.Store
	if s, exists := credentialStores[deployment.ID]; exists {
		store = s
	} else {
		opts := credentials.StoreOptions{
			AllowPlaintextPut:        true,
			DetectDefaultNativeStore: true,
		}
		if s, err := credentials.NewStore(DockerConfigPath(deployment), opts); err != nil {
			return nil, fmt.Errorf("could not create credentials store: %w", err)
		} else {
			store = s
			credentialStores[deployment.ID] = store
		}
	}

	if reg, err := remote.NewRegistry(agentenv.DistrRegistryHost); err != nil {
		return nil, err
	} else if previousJWT[deployment.ID] != jwt {
		if previousJWT[deployment.ID] != "" {
			if err := credentials.Logout(ctx, store, agentenv.DistrRegistryHost); err != nil {
				return nil, fmt.Errorf("docker logout failed for %v: %w", agentenv.DistrRegistryHost, err)
			}
			previousJWT[deployment.ID] = ""
		}

		reg.PlainHTTP = agentenv.DistrRegistryPlainHTTP
		if err := credentials.Login(ctx, store, reg, auth.Credential{Username: "-", Password: jwt}); err != nil {
			return nil, fmt.Errorf("docker login failed for %v: %w", agentenv.DistrRegistryHost, err)
		}

		previousJWT[deployment.ID] = jwt
	}

	if !maps.Equal(previousAuth[deployment.ID], deployment.RegistryAuth) {
		for url, registry := range deployment.RegistryAuth {
			if reg, err := remote.NewRegistry(url); err != nil {
				return nil, err
			} else {
				if err := credentials.Login(ctx, store, reg, auth.Credential{
					Username: registry.Username,
					Password: registry.Password,
				}); err != nil {
					return nil, fmt.Errorf("docker login failed for %v: %w", url, err)
				}
			}
		}
		for url := range previousAuth[deployment.ID] {
			if _, exists := deployment.RegistryAuth[url]; !exists {
				if err := credentials.Logout(ctx, store, url); err != nil {
					return nil, fmt.Errorf("docker logout failed for %v: %w", url, err)
				}
			}
		}
		previousAuth[deployment.ID] = deployment.RegistryAuth
	}

	return &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.DefaultCache,
		Credential: credentials.Credential(store),
	}, nil
}

func DeploymentTempDir(deployment api.AgentDeployment) string {
	return path.Join(os.TempDir(), deployment.ID.String())
}

func DockerConfigDir(deployment api.AgentDeployment) string {
	return path.Join(DeploymentTempDir(deployment), "docker")
}

func DockerConfigPath(deployment api.AgentDeployment) string {
	return path.Join(DockerConfigDir(deployment), dockerconfig.ConfigFileName)
}
