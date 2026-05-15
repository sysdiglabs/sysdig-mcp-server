package tools

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type ToolDiscoverRuntimeEventFieldValues struct {
	sysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewToolDiscoverRuntimeEventFieldValues(client sysdig.ExtendedClientWithResponsesInterface, clock clock.Clock) *ToolDiscoverRuntimeEventFieldValues {
	return &ToolDiscoverRuntimeEventFieldValues{
		sysdigClient: client,
		clock:        clock,
	}
}

func (h *ToolDiscoverRuntimeEventFieldValues) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	field := request.GetString("field", "")
	if field == "" {
		return mcp.NewToolResultErrorf("field is required"), nil
	}

	scopeHours := request.GetInt("scope_hours", 1)
	to := h.clock.Now()
	from := to.Add(-time.Duration(scopeHours) * time.Hour)

	params := &sysdig.GetEventFieldValuesParams{
		Field: field,
		From:  from.UnixNano(),
		To:    to.UnixNano(),
	}
	if filterExpr := request.GetString("filter_expr", ""); filterExpr != "" {
		params.Filter = &filterExpr
	}

	response, err := h.sysdigClient.GetEventFieldValuesWithResponse(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("error triggering request", err), nil
	}
	if response.StatusCode() >= 400 {
		return mcp.NewToolResultErrorf("error discovering field values, status code: %d, response: %s", response.StatusCode(), response.Body), nil
	}

	return mcp.NewToolResultJSON(response.JSON200)
}

func (h *ToolDiscoverRuntimeEventFieldValues) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("discover_runtime_event_field_values",
		mcp.WithDescription("Discover the distinct values of a runtime-events field present in a time window. Returns two buckets: `suggested` = values producing events in the window (fire order — what's actually happening); `other` = values known to the tenant but inactive in the window (catalog — what's possible). Use BEFORE writing a filter to learn which cluster / rule / image / namespace names are real, instead of guessing and getting empty results. Common fields to discover: kubernetes.cluster.name, kubernetes.namespace.name, ruleName, container.image.repo, host.hostName, aws.accountId, source, engine."),
		mcp.WithString("field",
			mcp.Description("Field whose distinct values to enumerate. Examples: kubernetes.cluster.name, ruleName, container.image.repo, host.hostName, severity, source, engine."),
			mcp.Required(),
			Examples(
				"kubernetes.cluster.name",
				"ruleName",
				"container.image.repo",
				"host.hostName",
				"severity",
				"source",
				"engine",
				"aws.accountId",
			),
		),
		mcp.WithNumber("scope_hours",
			mcp.Description("Number of hours back from now to scan. Maximum 336 (14 days). Default 1."),
			mcp.DefaultNumber(1),
		),
		mcp.WithString("filter_expr",
			mcp.Description("Optional filter expression to scope the search before enumerating values. Same DSL as other runtime-event tools. Without a filter, the enumeration spans all categories of events in the window."),
			Examples(
				`kubernetes.cluster.name = "production-gke"`,
				`engine = "machineLearning"`,
				`severity in ("0","1","2","3")`,
			),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("policy-events.read"),
	)

	s.AddTool(tool, h.handle)
}
