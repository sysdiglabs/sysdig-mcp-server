package tools

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolCountRuntimeEvents struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewToolCountRuntimeEvents(client sysdig.ExtendedClientWithResponsesInterface, clock clock.Clock) *ToolCountRuntimeEvents {
	return &ToolCountRuntimeEvents{
		sysdigClient: client,
		clock:        clock,
	}
}

func (h *ToolCountRuntimeEvents) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := toolRequestToCountParams(request, h.clock)

	response, err := h.sysdigClient.GetSecureEventsCountWithResponse(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("error triggering request", err), nil
	}
	if response.StatusCode() >= 400 {
		return mcp.NewToolResultErrorf("error counting events, status code: %d, response: %s", response.StatusCode(), response.Body), nil
	}

	return mcp.NewToolResultJSON(response.JSON200)
}

func toolRequestToCountParams(request mcp.CallToolRequest, clock clock.Clock) *sysdig.GetSecureEventsCountParams {
	scopeHours := request.GetInt("scope_hours", 1)
	to := clock.Now()
	from := to.Add(-time.Duration(scopeHours) * time.Hour)

	filter := composeSecureEventsFilter(request.GetString("filter_expr", ""))

	return &sysdig.GetSecureEventsCountParams{
		From:   from.UnixNano(),
		To:     to.UnixNano(),
		Filter: &filter,
	}
}

func (h *ToolCountRuntimeEvents) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("count_runtime_events",
		mcp.WithDescription("Count runtime security events matching a filter expression in the last N hours, without paginating event bodies. Returns a histogram across 16 event categories (policyEvents, scanningEvents, cloudTrailEvents, mlCloudEvents, oktaEvents, githubEvents, gcpEvents, falcoCloudEvents, admissionControllerEvents, profilingDetectionEvents, awsMlConsoleLoginEvents, hostScanningEvents, benchmarkEvents, complianceEvents, cloudsecEvents, statefulDetectionEvents) where each category carries a `countBySeverity` map keyed \"0\" (highest) through \"7\" (info). Use this when the question is \"how many\" rather than \"which ones\" — it is one call regardless of result size."),
		mcp.WithNumber("scope_hours",
			mcp.Description("Number of hours back from now to count events over. Maximum 336 (14 days) — the backend rejects wider windows. Default 1."),
			mcp.DefaultNumber(1),
		),
		mcp.WithString("filter_expr",
			mcp.Description(secureEventsFilterDSL),
			Examples(
				`severity in ("0","1","2","3")`,
				`ruleName = "Malware Detection"`,
				`kubernetes.cluster.name = "cluster1" and severity in ("0","1","2","3")`,
				`engine = "machineLearning"`,
				`aws.accountId = "123456789012"`,
			),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("policy-events.read"),
	)

	s.AddTool(tool, h.handle)
}
