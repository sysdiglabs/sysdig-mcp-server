package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolGenerateSysql struct {
	sysdigClient      sysdig.ExtendedClientWithResponsesInterface
	PermissionChecker PermissionChecker
}

func NewToolGenerateSysql(client sysdig.ExtendedClientWithResponsesInterface, checker PermissionChecker) *ToolGenerateSysql {
	return &ToolGenerateSysql{
		sysdigClient:      client,
		PermissionChecker: checker,
	}
}

func (h *ToolGenerateSysql) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	question := request.GetString("question", "")
	if question == "" {
		return mcp.NewToolResultError("question is required"), nil
	}

	response, err := h.sysdigClient.GenerateSysqlWithResponse(ctx, question)
	if err != nil {
		return mcp.NewToolResultError("error triggering request: " + err.Error()), nil
	}
	if response.HTTPResponse.StatusCode >= 400 {
		return mcp.NewToolResultErrorf("error generating SysQL query, status code: %d, response: %s", response.HTTPResponse.StatusCode, response.Body), nil
	}

	res, err := mcp.NewToolResultJSON(response.JSON200)
	if err != nil {
		return mcp.NewToolResultError("error parsing response: " + err.Error()), nil
	}

	return res, nil
}

func (h *ToolGenerateSysql) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("generate_sysql",
		mcp.WithDescription(`Generates a SysQL query from a natural language question.`),
		mcp.WithString("question",
			mcp.Description("A natural language question to be translated into a SysQL query."),
			mcp.Required(),
			Examples(
				`List all my containers with packages affected by vulnerabilities`,
				`Tell me the resources affected by any vulnerability that affects packages`,
				`Give me the vulnerabilities affecting images`,
			),
		),
		mcp.WithOutputSchema[map[string]any](),
	)
	s.AddTool(tool, h.handle)
}

func (h *ToolGenerateSysql) CanBeUsed() bool {
	return h.PermissionChecker.HasPermission("sage.exec")
}
