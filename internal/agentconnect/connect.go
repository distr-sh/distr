package agentconnect

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func buildURL(targetID uuid.UUID, org types.Organization, targetSecret string, preConnect bool) (string, error) {
	u, err := url.Parse(customdomains.AppDomainOrDefault(org))
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("targetId", targetID.String())
	query.Set("targetSecret", targetSecret)

	endpoint := "/api/v1/connect"
	if preConnect {
		endpoint = "/api/v1/pre-connect"
	}
	u = u.JoinPath(endpoint)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func BuildConnectURL(targetID uuid.UUID, org types.Organization, targetSecret string) (string, error) {
	return buildURL(targetID, org, targetSecret, false)
}

func BuildPreConnectURL(targetID uuid.UUID, org types.Organization, targetSecret string) (string, error) {
	return buildURL(targetID, org, targetSecret, true)
}

func GenerateConnectScript(targetID uuid.UUID, org types.Organization, targetSecret string) (string, error) {
	connectURL, err := BuildConnectURL(targetID, org, targetSecret)
	if err != nil {
		return "", fmt.Errorf("failed to build connect URL: %w", err)
	}

	var script strings.Builder
	script.WriteString("#!/bin/sh\n")

	if org.PreConnectScript != nil && strings.TrimSpace(*org.PreConnectScript) != "" {
		script.WriteString("# Pre-connect script\n")
		script.WriteString(*org.PreConnectScript)
		script.WriteString("\n\n")
	}

	script.WriteString("# Connect to Distr agent\n")
	script.WriteString(generateDockerConnectCommand(connectURL))

	if org.PostConnectScript != nil && strings.TrimSpace(*org.PostConnectScript) != "" {
		script.WriteString("\n\n# Post-connect script\n")
		script.WriteString(*org.PostConnectScript)
		script.WriteString("\n")
	}

	return script.String(), nil
}

func generateScriptCommand(scriptURL string, isSudo bool) string {
	shCmd := "sh"
	if isSudo {
		shCmd = "sudo " + shCmd
	}
	return fmt.Sprintf("curl -fsSL '%s' | %s", scriptURL, shCmd)
}

func generateDockerConnectCommand(connectURL string) string {
	return fmt.Sprintf("curl -fsSL '%s' | docker compose -f - up -d", connectURL)
}

func generateKubernetesConnectCommand(namespace string, connectURL string) string {
	return fmt.Sprintf("kubectl apply -n %s -f \"%s\"", namespace, connectURL)
}

func GenerateConnectCommand(
	deploymentTarget types.DeploymentTarget,
	org types.Organization,
	targetSecret string,
) (string, error) {
	if deploymentTarget.Type == types.DeploymentTypeDocker && org.HasFeature(types.FeaturePrePostScripts) {
		preConnectURL, err := BuildPreConnectURL(deploymentTarget.ID, org, targetSecret)
		if err != nil {
			return "", fmt.Errorf("failed to build pre-connect URL: %w", err)
		}
		return generateScriptCommand(preConnectURL, org.ConnectScriptIsSudo), nil
	}

	connectURL, err := BuildConnectURL(deploymentTarget.ID, org, targetSecret)
	if err != nil {
		return "", fmt.Errorf("failed to build connect URL: %w", err)
	}

	switch deploymentTarget.Type {
	case types.DeploymentTypeDocker:
		return generateDockerConnectCommand(connectURL), nil
	case types.DeploymentTypeKubernetes:
		if deploymentTarget.Namespace == nil {
			return "", fmt.Errorf("kubernetes deployment target must have a namespace")
		}
		return generateKubernetesConnectCommand(*deploymentTarget.Namespace, connectURL), nil
	case types.DeploymentTypeOpenTofu:
		return generateDockerConnectCommand(connectURL), nil
	default:
		return "", fmt.Errorf("unsupported deployment type: %s", deploymentTarget.Type)
	}
}
