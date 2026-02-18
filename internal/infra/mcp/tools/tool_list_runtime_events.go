package tools

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

const baseFilter = `source != "audittrail" and not originator in ("benchmarks","compliance","cloudsec","scanning","hostscanning")`

type ToolListRuntimeEvents struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewToolListRuntimeEvents(client sysdig.ExtendedClientWithResponsesInterface, clock clock.Clock) *ToolListRuntimeEvents {
	return &ToolListRuntimeEvents{
		sysdigClient: client,
		clock:        clock,
	}
}

func (h *ToolListRuntimeEvents) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := toolRequestToEventsV1Params(request, h.clock)

	response, err := h.sysdigClient.GetEventsV1WithResponse(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("error triggering request", err), nil
	}
	if response.StatusCode() >= 400 {
		return mcp.NewToolResultErrorf("error retrieving events, status code: %d, response: %s", response.StatusCode(), response.Body), nil
	}

	return mcp.NewToolResultJSON(response.JSON200)
}

func toolRequestToEventsV1Params(request mcp.CallToolRequest, clock clock.Clock) *sysdig.GetEventsV1Params {
	params := &sysdig.GetEventsV1Params{
		Limit: new(int32(request.GetInt("limit", 50))),
	}

	if cursor := request.GetString("cursor", ""); cursor != "" {
		params.Cursor = &cursor
	} else {
		scopeHours := request.GetInt("scope_hours", 1)
		to := clock.Now()
		from := to.Add(-time.Duration(scopeHours) * time.Hour)
		params.To = new(to.UnixNano())
		params.From = new(from.UnixNano())
	}

	params.Filter = new(baseFilter)
	if filterExpr := request.GetString("filter_expr", ""); filterExpr != "" {
		params.Filter = new(baseFilter + " and " + filterExpr)
	}

	return params
}

func (h *ToolListRuntimeEvents) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("list_runtime_events",
		mcp.WithDescription("List runtime security events from the last given hours, optionally filtered by severity level."),
		mcp.WithString("cursor",
			mcp.Description("Cursor for pagination."),
		),
		mcp.WithNumber("scope_hours",
			mcp.Description("Number of hours back from now to include events."),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of events to return."),
			mcp.DefaultNumber(50),
		),
		mcp.WithString("filter_expr",
			mcp.Description(`Logical filter expression to select runtime security events.
Supports operators: =, !=, in, contains, startsWith, exists.
Combine with and/or/not.
Key attributes include: severity (codes "0"-"7"), originator, sourceType, ruleName, rawEventCategory, kubernetes.cluster.name, host.hostName, container.imageName, aws.accountId, azure.subscriptionId, gcp.projectId, policyId, trigger.

You can specify the severity of the events based on the following cases:
- high-severity: 'severity in ("0","1","2","3")'
- medium: 'severity in ("4","5")'
- low: 'severity in ("6")'
- info: 'severity in ("7")'
`),
			Examples(
				`originator in ("awsCloudConnector","gcp") and not sourceType = "auditTrail"`,
				`ruleName contains "Login"`,
				`severity in ("0","1","2","3")`,
				`kubernetes.cluster.name = "cluster1"`,
				`host.hostName startsWith "web-"`,
				`container.imageName = "nginx:latest" and originator = "hostscanning"`,
				`aws.accountId = "123456789012"`,
				`policyId = "CIS_Docker_Benchmark"`,
			),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("policy-events.read"),
	)

	s.AddTool(tool, h.handle)
}
