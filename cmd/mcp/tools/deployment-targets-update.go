package tools

import (
	"context"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (m *Manager) NewUpdateDeploymentTargetTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"update_deployment_target",
			mcp.WithDescription("This tool updates an existing deployment target"),
			mcp.WithString("id", mcp.Required(), mcp.Description("ID of the deployment target to update")),
			mcp.WithString("name", mcp.Required()),
			mcp.WithString("type", mcp.Required(), mcp.Enum("docker", "kubernetes")),
			mcp.WithString("namespace"),
			mcp.WithString("scope", mcp.Enum("cluster", "namespace")),
			mcp.WithBoolean("metricsEnabled", mcp.DefaultBool(true)),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var deployment types.DeploymentTargetFull

			id, err := ParseUUID(request, "id")
			if err != nil {
				return mcp.NewToolResultErrorFromErr("id is invalid", err), nil
			}
			if id == uuid.Nil {
				return mcp.NewToolResultError("id is required"), nil
			}
			deployment.ID = id

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
							scope := types.DeploymentTargetScopeCluster
							deployment.Scope = &scope
						case string(types.DeploymentTargetScopeNamespace):
							scope := types.DeploymentTargetScopeNamespace
							deployment.Scope = &scope
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

			if result, err := m.client.DeploymentTargets().Update(ctx, deployment); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to update DeploymentTarget", err), nil
			} else {
				return JsonToolResult(result)
			}
		},
	}
}
