package mcp

import (
	"context"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolGetEventInfo struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewToolGetEventInfo(client sysdig.ExtendedClientWithResponsesInterface) *ToolGetEventInfo {
	return &ToolGetEventInfo{sysdigClient: client}
}

func (h *ToolGetEventInfo) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	eventId := request.GetString("event_id", "")
	if eventId == "" {
		return mcp.NewToolResultErrorf("event_id is required"), nil
	}

	response, err := h.sysdigClient.GetEventV1WithResponse(ctx, eventId)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("error triggering request", err), nil
	}
	if response.StatusCode() >= 400 {
		return mcp.NewToolResultErrorf("error retrieving event, status code: %d, response: %s", response.StatusCode(), response.Body), nil
	}

	return mcp.NewToolResultJSON(response.JSON200)
}

func (h *ToolGetEventInfo) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("get_event_info",
		mcp.WithDescription("Retrieve detailed information for a specific security event by its ID"),
		mcp.WithString("event_id",
			mcp.Description("The unique identifier of the security event."),
			mcp.Required(),
		),
		mcp.WithOutputSchema[map[string]any](),
	)

	s.AddTool(tool, h.handle)
}

func (h *ToolGetEventInfo) CanBeUsed() bool {
	permissions, err := h.sysdigClient.GetMyPermissionsWithResponse(context.Background())
	if err != nil {
		return false
	}

	return slices.Contains(permissions.JSON200.Permissions, "policy-events.read")
}
