package agentmanifest

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
	org types.Organization,
	secret *string,
) (io.Reader, error) {
	if tmpl, err := getTemplate(deploymentTarget); err != nil {
		return nil, err
	} else if data, err := getTemplateData(deploymentTarget, org, secret); err != nil {
		return nil, err
	} else {
		var buf bytes.Buffer
		return &buf, tmpl.Execute(&buf, data)
	}
}

func getTemplateData(
	deploymentTarget types.DeploymentTargetFull,
	org types.Organization,
	secret *string,
) (map[string]any, error) {
	var (
		loginEndpoint     string
		manifestEndpoint  string
		resourcesEndpoint string
		statusEndpoint    string
		metricsEndpoint   string
		logsEndpoint      string
		agentLogsEndpoint string
	)

	if u, err := url.Parse(customdomains.AppDomainOrDefault(org)); err != nil {
		return nil, err
	} else {
		u = u.JoinPath("api/v1/agent")
		loginEndpoint = u.JoinPath("login").String()
		manifestEndpoint = u.JoinPath("manifest").String()
		resourcesEndpoint = u.JoinPath("resources").String()
		statusEndpoint = u.JoinPath("status").String()
		metricsEndpoint = u.JoinPath("metrics").String()
		logsEndpoint = u.JoinPath("logs").String()
		agentLogsEndpoint = u.JoinPath("deployment-target-logs").String()
	}

	result := map[string]any{
		"agentDockerConfig": base64.StdEncoding.EncodeToString(env.AgentDockerConfig()),
		"agentInterval":     env.AgentInterval(),
		"agentVersion":      deploymentTarget.AgentVersion.Name,
		"agentVersionId":    deploymentTarget.AgentVersion.ID,
		"loginEndpoint":     loginEndpoint,
		"manifestEndpoint":  manifestEndpoint,
		"metricsEndpoint":   metricsEndpoint,
		"registryEnabled":   env.RegistryEnabled(),
		"registryHost":      customdomains.RegistryDomainOrDefault(org),
		"registryPlainHttp": buildconfig.IsDevelopment(),
		"resourcesEndpoint": resourcesEndpoint,
		"statusEndpoint":    statusEndpoint,
		"targetId":          deploymentTarget.ID,
		"targetSecret":      secret,
		"logsEndpoint":      logsEndpoint,
		"agentLogsEndpoint": agentLogsEndpoint,
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
	if deploymentTarget.Type == types.DeploymentTypeDocker {
		return resources.GetTemplate(path.Join(
			"agent/docker",
			deploymentTarget.AgentVersion.ComposeFileRevision,
			"docker-compose.yaml.tmpl",
		))
	} else {
		return resources.GetTemplate(path.Join(
			"agent/kubernetes",
			deploymentTarget.AgentVersion.ManifestFileRevision,
			"manifest.yaml.tmpl",
		))
	}
}
