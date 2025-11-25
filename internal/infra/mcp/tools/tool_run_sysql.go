package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolRunSysql struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewToolRunSysql(client sysdig.ExtendedClientWithResponsesInterface) *ToolRunSysql {
	return &ToolRunSysql{
		sysdigClient: client,
	}
}

func (h *ToolRunSysql) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sysqlQuery := request.GetString("sysql_query", "")
	if sysqlQuery == "" {
		return mcp.NewToolResultErrorf("sysql_query is required"), nil
	}

	// Ensure the query ends with a semicolon
	if !strings.HasSuffix(strings.TrimSpace(sysqlQuery), ";") {
		sysqlQuery = sysqlQuery + ";"
	}

	// Create the request body
	body := sysdig.QuerySysqlPostJSONRequestBody{
		Q: sysqlQuery,
	}

	response, err := h.sysdigClient.QuerySysqlPostWithResponse(ctx, body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("error triggering request", err), nil
	}
	if response.StatusCode() >= 400 {
		return mcp.NewToolResultErrorf("error retrieving SysQL results, status code: %d, response: %s", response.StatusCode(), response.Body), nil
	}

	return mcp.NewToolResultJSON(response.JSON200)
}

func (h *ToolRunSysql) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("run_sysql",
		mcp.WithDescription(`Execute a SysQL query directly against the Sysdig API. You should try generating a SysQL query first to ensure that it's valid.`),
		mcp.WithString("sysql_query",
			mcp.Description("A valid SysQL query string to execute directly."),
			mcp.Required(),
			Examples(
				`MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10`,
				`MATCH KubeWorkload AS k AFFECTED_BY Vulnerability WHERE k.namespace = 'production'`,
				`MATCH CloudResource WHERE type = 'aws_s3_bucket' RETURN *`,
				`MATCH Vulnerability AS v WHERE v.name = 'CVE-2024-1234' RETURN v`,
			),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		WithRequiredPermissions("sage.exec", "risks.read"),
	)
	s.AddTool(tool, h.handle)
}
