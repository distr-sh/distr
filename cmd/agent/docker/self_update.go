package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/moby/moby/api/types/container"
	mobyClient "github.com/moby/moby/client"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// updateContainerName is the fixed name of the updater container, used to enforce single-flight.
const updateContainerName = "distr-agent-update"

// containerIDPattern matches the agent's own container ID in /proc/self/mountinfo. Docker mounts
// /etc/hostname etc. from .../containers/<id>/ regardless of network mode, so this works even with
// host networking (where the hostname is the host's and can't identify the container).
var containerIDPattern = regexp.MustCompile(`containers/([0-9a-f]{64})/`)

func RunAgentSelfUpdate(ctx context.Context) error {
	if manifest, err := client.Manifest(ctx); err != nil {
		return fmt.Errorf("error fetching agent manifest: %w", err)
	} else if parsedManifest, err := DecodeComposeFile(manifest); err != nil {
		return fmt.Errorf("error parsing agent manifest: %w", err)
	} else if err := PatchAgentManifest(parsedManifest); err != nil {
		return fmt.Errorf("error patching agent manifest: %w", err)
	} else if err := ApplyAgentComposeFile(ctx, parsedManifest); err != nil {
		return fmt.Errorf("error applying agent manifest: %w", err)
	} else {
		return nil
	}
}

func PatchAgentManifest(manifest map[string]any) error {
	if svcs, ok := manifest["services"].(map[string]any); ok {
		if svc, ok := svcs["agent"].(map[string]any); ok {
			if env, ok := svc["environment"].(map[string]any); ok {
				env["DISTR_TARGET_SECRET"] = os.Getenv("DISTR_TARGET_SECRET")
			} else {
				return errors.New("env is not an object")
			}
		} else {
			return errors.New("service \"agent\" is not an object")
		}
	} else {
		return errors.New("services is not an object")
	}
	return nil
}

func GetAgentImageFromManifest(manifest map[string]any) (string, error) {
	if svcs, ok := manifest["services"].(map[string]any); ok {
		if svc, ok := svcs["agent"].(map[string]any); ok {
			if image, ok := svc["image"].(string); ok {
				return image, nil
			} else {
				return "", errors.New("image is not a string")
			}
		} else {
			return "", errors.New("service \"agent\" is not an object")
		}
	} else {
		return "", errors.New("services is not an object")
	}
}

// ApplyAgentComposeFile runs the agent self-update in a separate, detached docker container.
// Running it from the agent directly would never finish, leaving the installation broken.
//
// The update is single-flight: the detached "docker run" returns immediately while the old agent
// keeps running, so without a guard every tick would start another updater, causing concurrent
// "docker compose up" runs and renamed (hex-prefixed) leftover containers.
func ApplyAgentComposeFile(ctx context.Context, manifest map[string]any) error {
	if running, err := selfUpdateInProgress(ctx); err != nil {
		return fmt.Errorf("could not determine whether a self-update is already running: %w", err)
	} else if running {
		logger.Info("self-update is already in progress, skipping")
		return nil
	}

	containerID, err := currentContainerID()
	if err != nil {
		return fmt.Errorf("could not determine own container ID: %w", err)
	}

	// I tried using something like "echo ... | base64 -d | docker compose ...", but I kept getting
	// "filename too long" errors with that approach.
	// It is therefore necessary to write the docker-compose.yaml data to a file instead.
	// Because of how DinD works, this file, which is also mounted in the new container must be
	// either on the host filesystem or in a shared volume.
	file, err := os.Create(path.Join(ScratchDir(), "distr-update.yaml"))
	if err != nil {
		return err
	}
	if err := yaml.NewEncoder(file).Encode(manifest); err != nil {
		file.Close()
		return err
	}
	file.Close()

	// The self-update container uses the same image as the new agent.
	// This should save some time and disk space on the host, but it means that we have to be
	// careful about migrating to a different base image for the agent.
	imageName, err := GetAgentImageFromManifest(manifest)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx,
		"docker", "run", "--detach", "--rm",
		"--name", updateContainerName,
		"--entrypoint", "/usr/local/bin/docker-entrypoint.sh",
		"--env", "HOST_DOCKER_CONFIG_DIR="+os.Getenv("HOST_DOCKER_CONFIG_DIR"),
		"--volumes-from", containerID,
		imageName,
		"docker", "compose", "-f", file.Name(), "up", "-d",
	)
	out, err := cmd.CombinedOutput()
	logger.Sugar().Infof("self-update output: %v", strings.TrimSpace(string(out)))
	return err
}

// selfUpdateInProgress reports whether the updater container is currently running. Non-running
// leftovers with the same name are removed so a new updater's "--name" does not collide.
func selfUpdateInProgress(ctx context.Context) (bool, error) {
	apiClient := dockerCli.Client()
	result, err := apiClient.ContainerList(ctx, mobyClient.ContainerListOptions{
		All:     true,
		Filters: mobyClient.Filters{}.Add("name", "^/"+updateContainerName+"$"),
	})
	if err != nil {
		return false, err
	}

	running := false
	for _, c := range result.Items {
		if c.State == container.StateRunning {
			running = true
			continue
		}
		if _, err := apiClient.ContainerRemove(ctx, c.ID, mobyClient.ContainerRemoveOptions{Force: true}); err != nil {
			logger.Warn("failed to remove stale self-update container", zap.String("id", c.ID), zap.Error(err))
		}
	}
	return running, nil
}

// currentContainerID returns the ID of the agent's own container from /proc/self/mountinfo. This
// avoids a hardcoded name, which is unreliable because docker may rename containers with a hex
// prefix, e.g. "611d096d207d_distr-agent-1".
func currentContainerID() (string, error) {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	if match := containerIDPattern.FindSubmatch(data); match != nil {
		return string(match[1]), nil
	}
	return "", errors.New("no container ID found in /proc/self/mountinfo")
}
