package mcp

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolGetEventProcessTree struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewToolGetEventProcessTree(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *ToolGetEventProcessTree {
	return &ToolGetEventProcessTree{sysdigClient: sysdigClient}
}

func (h *ToolGetEventProcessTree) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	eventID := request.GetString("event_id", "")
	if eventID == "" {
		return mcp.NewToolResultErrorf("event_id is required"), nil
	}

	branchesResp, branchesErr := h.sysdigClient.GetProcessTreeBranchesWithResponse(ctx, eventID)
	if branchesErr != nil && !errors.Is(branchesErr, sysdig.ErrNotFound) {
		return mcp.NewToolResultErrorFromErr("error requesting process tree branches", branchesErr), nil
	}

	treesResp, treesErr := h.sysdigClient.GetProcessTreeTreesWithResponse(ctx, eventID)
	if treesErr != nil && !errors.Is(treesErr, sysdig.ErrNotFound) {
		return mcp.NewToolResultErrorFromErr("error requesting process tree trees", treesErr), nil
	}

	if errors.Is(branchesErr, sysdig.ErrNotFound) || errors.Is(treesErr, sysdig.ErrNotFound) {
		return mcp.NewToolResultJSON(map[string]any{
			"branches": map[string]any{},
			"tree":     map[string]any{},
			"metadata": map[string]any{
				"note": "Process tree not available for this event",
			},
		})
	}

	result := map[string]any{
		"branches": *branchesResp.JSON200,
		"tree":     *treesResp.JSON200,
	}

	return mcp.NewToolResultJSON(result)
}

func (h *ToolGetEventProcessTree) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("get_event_process_tree",
		mcp.WithDescription("Retrieves the process tree for a specific security event.\nNot every event has a process tree, so this may return an empty tree."),
		mcp.WithString("event_id",
			mcp.Description("The unique identifier of the security event."),
			mcp.Required(),
		),
		mcp.WithOutputSchema[map[string]any](),
	)

	s.AddTool(tool, h.handle)
}
