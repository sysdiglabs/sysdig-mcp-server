package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type K8sListTopMemoryConsumedWorkload struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewK8sListTopMemoryConsumedWorkload(sysdigClient sysdig.ExtendedClientWithResponsesInterface, clk clock.Clock) *K8sListTopMemoryConsumedWorkload {
	return &K8sListTopMemoryConsumedWorkload{
		SysdigClient: sysdigClient,
		clock:        clk,
	}
}

func (t *K8sListTopMemoryConsumedWorkload) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("k8s_list_top_memory_consumed_workload",
		mcp.WithDescription("Lists memory-intensive workloads (all containers). Optionally pass start/end (RFC3339) to query a historical window (averaged over the window) instead of the current instant snapshot."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_type", mcp.Description("The type of the workload to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of workloads to return."),
			mcp.DefaultNumber(20),
		),
		WithTimeWindowParams(),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *K8sListTopMemoryConsumedWorkload) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	limit := mcp.ParseInt(request, "limit", 20)

	tw, err := ParseTimeWindow(request, t.clock)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid time window", err), nil
	}

	query := buildTopMemoryConsumedByWorkloadQuery(clusterName, namespaceName, workloadType, workloadName, limit, tw)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}
	if err := tw.ApplyToParams(params); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to build eval time", err), nil
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get workload list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get workload list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildTopMemoryConsumedByWorkloadQuery(clusterName, namespaceName, workloadType, workloadName string, limit int, tw TimeWindow) string {
	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf("kube_cluster_name=\"%s\"", clusterName))
	}
	if namespaceName != "" {
		filters = append(filters, fmt.Sprintf("kube_namespace_name=\"%s\"", namespaceName))
	}
	if workloadType != "" {
		filters = append(filters, fmt.Sprintf("kube_workload_type=\"%s\"", workloadType))
	}
	if workloadName != "" {
		filters = append(filters, fmt.Sprintf("kube_workload_name=\"%s\"", workloadName))
	}

	filterString := ""
	if len(filters) > 0 {
		filterString = fmt.Sprintf("{%s}", strings.Join(filters, ","))
	}

	metric := fmt.Sprintf("sysdig_container_memory_used_bytes%s", filterString)
	if !tw.IsZero() {
		metric = fmt.Sprintf("avg_over_time(%s%s)", metric, tw.RangeSelector())
	}

	innerQuery := fmt.Sprintf("sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name) (%s)", metric)
	return fmt.Sprintf("topk(%d, %s)", limit, innerQuery)
}
