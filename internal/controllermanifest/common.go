package controllermanifest

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/url"
	"path"
	"text/template"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/resources"
	"github.com/distr-sh/distr/internal/types"
)

func Get(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	org types.OrganizationWithBranding,
	secret *string,
) (io.Reader, error) {
	if tmpl, err := getTemplate(deploymentTarget); err != nil {
		return nil, err
	} else if data, err := getTemplateData(ctx, deploymentTarget, org, secret); err != nil {
		return nil, err
	} else {
		var buf bytes.Buffer
		return &buf, tmpl.Execute(&buf, data)
	}
}

func getTemplateData(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	org types.OrganizationWithBranding,
	secret *string,
) (map[string]any, error) {
	var (
		loginEndpoint             string
		manifestEndpoint          string
		resourcesEndpoint         string
		statusEndpoint            string
		metricsEndpoint           string
		deploymentMetricsEndpoint string
		logsEndpoint              string
		controllerLogsEndpoint    string
	)

	if u, err := url.Parse(customdomains.ControllerDomainOrDefault(ctx, org.ID, org.Branding)); err != nil {
		return nil, err
	} else {
		u = u.JoinPath("api/v1/controller")
		loginEndpoint = u.JoinPath("login").String()
		manifestEndpoint = u.JoinPath("manifest").String()
		resourcesEndpoint = u.JoinPath("resources").String()
		statusEndpoint = u.JoinPath("status").String()
		metricsEndpoint = u.JoinPath("metrics").String()
		deploymentMetricsEndpoint = u.JoinPath("deployments").String()
		logsEndpoint = u.JoinPath("logs").String()
		controllerLogsEndpoint = u.JoinPath("deployment-target-logs").String()
	}

	result := map[string]any{
		"controllerDockerConfig":    base64.StdEncoding.EncodeToString(env.ControllerDockerConfig()),
		"controllerInterval":        env.ControllerInterval(),
		"controllerVersion":         deploymentTarget.ControllerVersion.Name,
		"controllerVersionId":       deploymentTarget.ControllerVersion.ID,
		"autohealAll":               deploymentTarget.AutohealEnabled,
		"loginEndpoint":             loginEndpoint,
		"manifestEndpoint":          manifestEndpoint,
		"metricsEndpoint":           metricsEndpoint,
		"deploymentMetricsEndpoint": deploymentMetricsEndpoint,
		"registryEnabled":           env.RegistryEnabled(),
		"registryHost":              customdomains.RegistryDomainOrDefault(ctx, org.ID, org.Branding),
		"registryPlainHttp":         buildconfig.IsDevelopment(),
		"resourcesEndpoint":         resourcesEndpoint,
		"statusEndpoint":            statusEndpoint,
		"targetId":                  deploymentTarget.ID,
		"targetSecret":              secret,
		"logsEndpoint":              logsEndpoint,
		"controllerLogsEndpoint":    controllerLogsEndpoint,
		"metricsEnabled":            deploymentTarget.MetricsEnabled,
		"dockerSocketPath":          deploymentTarget.DockerSocketPath(),
	}
	if deploymentTarget.Namespace != nil {
		result["targetNamespace"] = *deploymentTarget.Namespace
	}
	if deploymentTarget.Scope != nil {
		result["targetScope"] = *deploymentTarget.Scope
	}
	if deploymentTarget.Resources != nil {
		result["targetResources"] = deploymentTarget.Resources
	}
	return result, nil
}

func getTemplate(deploymentTarget types.DeploymentTargetFull) (*template.Template, error) {
	manifestRevision := deploymentTarget.ControllerVersion.ManifestFileRevision
	composeRevision := deploymentTarget.ControllerVersion.ComposeFileRevision
	if deploymentTarget.LegacyAgentManifest {
		if manifestRevision == types.CurrentManifestFileRevision {
			manifestRevision = types.LegacyManifestFileRevision
		}
		if composeRevision == types.CurrentComposeFileRevision {
			composeRevision = types.LegacyComposeFileRevision
		}
	}
	if deploymentTarget.Type == types.DeploymentTypeDocker {
		return resources.GetTemplate(path.Join("controller/docker", composeRevision, "docker-compose.yaml.tmpl"))
	} else {
		return resources.GetTemplate(path.Join("controller/kubernetes", manifestRevision, "manifest.yaml.tmpl"))
	}
}
