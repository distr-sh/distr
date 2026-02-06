package tools

import (
	"context"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (m *Manager) NewCreateDeploymentTargetTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"create_deployment_target",
			mcp.WithDescription("This tools creates a new deployment target"),
			mcp.WithString("name", mcp.Required()),
			mcp.WithString("type", mcp.Required(), mcp.Enum("docker", "kubernetes")),
			mcp.WithString("namespace"),
			mcp.WithString("scope", mcp.Enum("cluster", "namespace")),
			mcp.WithBoolean("metricsEnabled", mcp.DefaultBool(true)),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var deployment types.DeploymentTargetFull

			if name := mcp.ParseString(request, "name", ""); name == "" {
				return mcp.NewToolResultError("name is required"), nil
			} else {
				deployment.Name = name
			}

			if ts := mcp.ParseString(request, "type", ""); ts == "" {
				return mcp.NewToolResultError("type is required"), nil
			} else {
				switch ts {
				case string(types.DeploymentTypeKubernetes):
					deployment.Type = types.DeploymentTypeKubernetes
					if ns := mcp.ParseString(request, "namespace", ""); ns == "" {
						return mcp.NewToolResultError("namespace is required if type is kubernetes"), nil
					} else {
						deployment.Namespace = &ns
					}
					if scope := mcp.ParseString(request, "scope", ""); scope == "" {
						return mcp.NewToolResultError("scope is required if type is kubernetes"), nil
					} else {
						switch scope {
						case string(types.DeploymentTargetScopeCluster):
							deployment.Scope = util.PtrTo(types.DeploymentTargetScopeCluster)
						case string(types.DeploymentTargetScopeNamespace):
							deployment.Scope = util.PtrTo(types.DeploymentTargetScopeNamespace)
						default:
							return mcp.NewToolResultError("scope must be either cluster or namespace"), nil
						}
					}
				case string(types.DeploymentTypeDocker):
					deployment.Type = types.DeploymentTypeDocker
				default:
					return mcp.NewToolResultError("type must be either docker or kubernetes"), nil
				}
			}

			deployment.MetricsEnabled = mcp.ParseBoolean(request, "metricsEnabled", true)

			if result, err := m.client.DeploymentTargets().Create(ctx, deployment); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to create DeploymentTarget", err), nil
			} else {
				return JsonToolResult(result)
			}
		},
	}
}
