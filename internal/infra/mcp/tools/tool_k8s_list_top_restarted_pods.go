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

type K8sListTopRestartedPods struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewK8sListTopRestartedPods(sysdigClient sysdig.ExtendedClientWithResponsesInterface, clk clock.Clock) *K8sListTopRestartedPods {
	return &K8sListTopRestartedPods{
		SysdigClient: sysdigClient,
		clock:        clk,
	}
}

func (t *K8sListTopRestartedPods) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("k8s_list_top_restarted_pods",
		mcp.WithDescription("Lists the pods with the highest number of container restarts in the specified scope (cluster, namespace, workload, or individual pod). By default, it returns the top 10. Optionally pass start/end (RFC3339) to count restarts that occurred *within* the window (via increase() on the counter) instead of total lifetime restarts."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_type", mcp.Description("The type of the workload to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithString("pod_name", mcp.Description("The name of the pod to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pods to return."),
			mcp.DefaultNumber(10),
		),
		WithTimeWindowParams(),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *K8sListTopRestartedPods) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	podName := mcp.ParseString(request, "pod_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	tw, err := ParseTimeWindow(request, t.clock)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid time window", err), nil
	}
	evalTime, err := tw.EvalTime()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to build eval time", err), nil
	}

	query := buildKubeTopRestartsQuery(clusterName, namespaceName, workloadType, workloadName, podName, limit, tw)

	params := &sysdig.GetQueryV1Params{
		Query: query,
		Time:  evalTime,
	}
	if !tw.IsZero() {
		timeout := sysdig.Timeout(windowedQueryTimeout)
		params.Timeout = &timeout
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get pod list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get pod list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildKubeTopRestartsQuery(clusterName, namespaceName, workloadType, workloadName, podName string, limit int, tw TimeWindow) string {
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
	if podName != "" {
		filters = append(filters, fmt.Sprintf("kube_pod_name=\"%s\"", podName))
	}

	filterString := ""
	if len(filters) > 0 {
		filterString = "{" + strings.Join(filters, ",") + "}"
	}

	metric := fmt.Sprintf("kube_pod_container_status_restarts_total%s", filterString)
	if !tw.IsZero() {
		metric = fmt.Sprintf("increase(%s%s)", metric, tw.RangeSelector())
	}

	return fmt.Sprintf("topk(%d, sum by(pod, kube_cluster_name, kube_namespace_name) (%s) > 0)", limit, metric)
}
