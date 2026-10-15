package controllerconnect

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func buildURL(
	ctx context.Context,
	targetID uuid.UUID,
	org types.OrganizationWithBranding,
	targetSecret string,
	preConnect bool,
) (string, error) {
	u, err := url.Parse(customdomains.ControllerDomainOrDefault(ctx, org.ID, org.Branding))
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

func BuildConnectURL(
	ctx context.Context,
	targetID uuid.UUID,
	org types.OrganizationWithBranding,
	targetSecret string,
) (string, error) {
	return buildURL(ctx, targetID, org, targetSecret, false)
}

func BuildPreConnectURL(
	ctx context.Context,
	targetID uuid.UUID,
	org types.OrganizationWithBranding,
	targetSecret string,
) (string, error) {
	return buildURL(ctx, targetID, org, targetSecret, true)
}

// usesConnectScript reports whether the connect command of the deployment target has to fetch the
// connect script rather than apply the controller manifest directly. Sudo is a reason on its own
// for Docker, since it is the only way to run docker compose as root.
func usesConnectScript(deploymentTarget types.DeploymentTarget, org types.OrganizationWithBranding) bool {
	if !org.HasFeature(types.FeaturePrePostScripts) {
		return false
	}
	pre, post := org.ConnectScripts(deploymentTarget.Type)
	if pre != "" || post != "" {
		return true
	}
	return deploymentTarget.Type == types.DeploymentTypeDocker && org.ConnectScriptIsSudo
}

func GenerateConnectScript(
	ctx context.Context,
	deploymentTarget types.DeploymentTarget,
	org types.OrganizationWithBranding,
	targetSecret string,
) (string, error) {
	connectURL, err := BuildConnectURL(ctx, deploymentTarget.ID, org, targetSecret)
	if err != nil {
		return "", fmt.Errorf("failed to build connect URL: %w", err)
	}
	return renderConnectScript(deploymentTarget, org, connectURL)
}

func renderConnectScript(
	deploymentTarget types.DeploymentTarget,
	org types.OrganizationWithBranding,
	connectURL string,
) (string, error) {
	connectCommand, err := manifestConnectCommand(deploymentTarget, connectURL)
	if err != nil {
		return "", err
	}

	preConnectScript, postConnectScript := org.ConnectScripts(deploymentTarget.Type)

	var script strings.Builder
	script.WriteString("#!/bin/sh")

	if deploymentTarget.Namespace != nil {
		script.WriteString("\n\n")
		fmt.Fprintf(&script, "export DISTR_NAMESPACE=%s", shellQuote(*deploymentTarget.Namespace))
	}

	if preConnectScript != "" {
		script.WriteString("\n\n")
		script.WriteString("# Pre-connect script\n")
		script.WriteString(preConnectScript)
	}

	script.WriteString("\n\n")
	script.WriteString("# Connect to Distr controller\n")
	script.WriteString(connectCommand)

	if postConnectScript != "" {
		script.WriteString("\n\n")
		script.WriteString("# Post-connect script\n")
		script.WriteString(postConnectScript)
	}

	script.WriteString("\n")

	return script.String(), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func generateScriptCommand(scriptURL string, isSudo bool) string {
	shCmd := "sh"
	if isSudo {
		shCmd = "sudo " + shCmd
	}
	return fmt.Sprintf("curl -fsSL %s | %s", shellQuote(scriptURL), shCmd)
}

func generateDockerConnectCommand(connectURL string) string {
	return fmt.Sprintf("curl -fsSL %s | docker compose -f - up -d", shellQuote(connectURL))
}

func generateKubernetesConnectCommand(namespace string, connectURL string) string {
	return fmt.Sprintf("kubectl apply -n %s -f %s", shellQuote(namespace), shellQuote(connectURL))
}

func manifestConnectCommand(deploymentTarget types.DeploymentTarget, connectURL string) (string, error) {
	switch deploymentTarget.Type {
	case types.DeploymentTypeDocker:
		return generateDockerConnectCommand(connectURL), nil
	case types.DeploymentTypeKubernetes:
		if deploymentTarget.Namespace == nil {
			return "", fmt.Errorf("kubernetes deployment target must have a namespace")
		}
		return generateKubernetesConnectCommand(*deploymentTarget.Namespace, connectURL), nil
	default:
		return "", fmt.Errorf("unsupported deployment type: %s", deploymentTarget.Type)
	}
}

func GenerateConnectCommand(
	ctx context.Context,
	deploymentTarget types.DeploymentTarget,
	org types.OrganizationWithBranding,
	targetSecret string,
) (string, error) {
	if usesConnectScript(deploymentTarget, org) {
		preConnectURL, err := BuildPreConnectURL(ctx, deploymentTarget.ID, org, targetSecret)
		if err != nil {
			return "", fmt.Errorf("failed to build pre-connect URL: %w", err)
		}
		return generateScriptCommand(
			preConnectURL,
			deploymentTarget.Type == types.DeploymentTypeDocker && org.ConnectScriptIsSudo,
		), nil
	}

	connectURL, err := BuildConnectURL(ctx, deploymentTarget.ID, org, targetSecret)
	if err != nil {
		return "", fmt.Errorf("failed to build connect URL: %w", err)
	}
	return manifestConnectCommand(deploymentTarget, connectURL)
}
