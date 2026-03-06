package tools

import (
	"context"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (m *Manager) NewListApplicationEntitlementsTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"list_application_entitlements",
			mcp.WithDescription("This tool retrieves all application entitlements"),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if entitlements, err := m.client.ApplicationEntitlements().List(ctx); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to list Application Entitlements", err), nil
			} else {
				return JsonToolResult(entitlements)
			}
		},
	}
}

func (m *Manager) NewGetApplicationEntitlementTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"get_application_entitlement",
			mcp.WithDescription("This tool retrieves a specific application entitlement"),
			mcp.WithString("id", mcp.Required(), mcp.Description("ID of the entitlement to retrieve")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := ParseUUID(request, "id")
			if err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to parse entitlement ID", err), nil
			}
			if id == uuid.Nil {
				return mcp.NewToolResultError("ID is required"), nil
			}

			if entitlement, err := m.client.ApplicationEntitlements().Get(ctx, id); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to get Application Entitlement", err), nil
			} else {
				return JsonToolResult(entitlement)
			}
		},
	}
}

func (m *Manager) NewCreateApplicationEntitlementTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"create_application_entitlement",
			mcp.WithDescription("This tool creates a new application entitlement"),
			mcp.WithObject("entitlement", mcp.Required(), mcp.Description("Entitlement object to create")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			entitlement, err := ParseT[*types.ApplicationEntitlementWithVersions](request, "entitlement", nil)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to parse entitlement data", err), nil
			}

			if result, err := m.client.ApplicationEntitlements().Create(ctx, entitlement); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to create Application Entitlement", err), nil
			} else {
				return JsonToolResult(result)
			}
		},
	}
}

func (m *Manager) NewUpdateApplicationEntitlementTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"update_application_entitlement",
			mcp.WithDescription("This tool updates an existing application entitlement"),
			mcp.WithObject("entitlement", mcp.Required(), mcp.Description("Entitlement object to update")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			entitlement, err := ParseT[*types.ApplicationEntitlementWithVersions](request, "entitlement", nil)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to parse entitlement data", err), nil
			}

			if result, err := m.client.ApplicationEntitlements().Update(ctx, entitlement); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to update Application Entitlement", err), nil
			} else {
				return JsonToolResult(result)
			}
		},
	}
}

func (m *Manager) NewDeleteApplicationEntitlementTool() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(
			"delete_application_entitlement",
			mcp.WithDescription("This tool deletes an application entitlement"),
			mcp.WithString("id", mcp.Required(), mcp.Description("ID of the entitlement to delete")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := ParseUUID(request, "id")
			if err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to parse entitlement ID", err), nil
			}
			if id == uuid.Nil {
				return mcp.NewToolResultError("ID is required"), nil
			}

			if err := m.client.ApplicationEntitlements().Delete(ctx, id); err != nil {
				return mcp.NewToolResultErrorFromErr("Failed to delete Application Entitlement", err), nil
			}
			return JsonToolResult(map[string]string{"status": "Application Entitlement deleted successfully"})
		},
	}
}
