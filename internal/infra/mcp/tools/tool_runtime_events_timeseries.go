package tools

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolRuntimeEventsTimeseries struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewToolRuntimeEventsTimeseries(client sysdig.ExtendedClientWithResponsesInterface, clock clock.Clock) *ToolRuntimeEventsTimeseries {
	return &ToolRuntimeEventsTimeseries{
		sysdigClient: client,
		clock:        clock,
	}
}

func (h *ToolRuntimeEventsTimeseries) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := toolRequestToTimeseriesParams(request, h.clock)

	response, err := h.sysdigClient.GetSecureEventsTimeseriesByWithResponse(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("error triggering request", err), nil
	}
	if response.StatusCode() >= 400 {
		return mcp.NewToolResultErrorf("error retrieving event timeseries, status code: %d, response: %s", response.StatusCode(), response.Body), nil
	}

	return mcp.NewToolResultJSON(response.JSON200)
}

func toolRequestToTimeseriesParams(request mcp.CallToolRequest, clock clock.Clock) *sysdig.GetSecureEventsTimeseriesByParams {
	scopeHours := request.GetInt("scope_hours", 1)
	to := clock.Now()
	from := to.Add(-time.Duration(scopeHours) * time.Hour)

	filter := composeSecureEventsFilter(request.GetString("filter_expr", ""))

	return &sysdig.GetSecureEventsTimeseriesByParams{
		From:   from.UnixNano(),
		To:     to.UnixNano(),
		Field:  request.GetString("field", "severity"),
		Rows:   int32(request.GetInt("rows", 60)),
		Limit:  int32(request.GetInt("limit", 50)),
		Filter: &filter,
	}
}

func (h *ToolRuntimeEventsTimeseries) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("runtime_events_timeseries",
		mcp.WithDescription("Bucket runtime security event counts over time, grouped by a categorical field (default \"severity\"). Use to locate when a burst started or ended without paginating event bodies. Returns a nested structure: data.subCounts[<group-value>].subCounts.timestamp.subCounts[<bucket-ns>].count, plus a top-level `step` giving the bucket width in nanoseconds. The server picks the coarsest bucket size that fits the `rows` upper bound; minimum bucket size is 1 minute. Two-call boundary-finding pattern: first call with rows=1000 over a wide window to bracket the burst, second call with rows=3600 over the bracketed range to drill to 1-minute resolution."),
		mcp.WithNumber("scope_hours",
			mcp.Description("Number of hours back from now to bucket events over. Maximum 336 (14 days). Default 1."),
			mcp.DefaultNumber(1),
		),
		mcp.WithString("field",
			mcp.Description("Categorical field to group by. \"severity\" is the useful default for triage; other categorical fields are also accepted by the backend."),
			mcp.DefaultString("severity"),
		),
		mcp.WithNumber("rows",
			mcp.Description("Upper bound on the number of time buckets. Server picks the coarsest step (minimum 1 minute) that yields <= rows buckets. Use ~1000 for a coarse pass over a wide window, then ~3600 over a bracketed range to force 1-minute buckets."),
			mcp.DefaultNumber(60),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of distinct values reported under the chosen field. The backend requires it; 50 is the canonical default and is never a bottleneck for severity (8 codes)."),
			mcp.DefaultNumber(50),
		),
		mcp.WithString("filter_expr",
			mcp.Description(secureEventsFilterDSL),
			Examples(
				`severity in ("0","1","2","3")`,
				`ruleName = "Suspicious Outbound Connection"`,
				`kubernetes.cluster.name = "cluster1"`,
				`engine = "machineLearning"`,
			),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("policy-events.read"),
	)

	s.AddTool(tool, h.handle)
}
