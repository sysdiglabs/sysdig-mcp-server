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

type K8sListUnderutilizedPodsCPUQuota struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewK8sListUnderutilizedPodsCPUQuota(sysdigClient sysdig.ExtendedClientWithResponsesInterface, clk clock.Clock) *K8sListUnderutilizedPodsCPUQuota {
	return &K8sListUnderutilizedPodsCPUQuota{
		SysdigClient: sysdigClient,
		clock:        clk,
	}
}

func (t *K8sListUnderutilizedPodsCPUQuota) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("k8s_list_underutilized_pods_cpu_quota",
		mcp.WithDescription("List Kubernetes pods with CPU usage below 25% of the quota limit. Optionally pass start/end (RFC3339) to evaluate the ratio averaged over a historical window instead of the current instant snapshot."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
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

func (t *K8sListUnderutilizedPodsCPUQuota) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	tw, err := ParseTimeWindow(request, t.clock)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid time window", err), nil
	}
	evalTime, err := tw.EvalTime()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to build eval time", err), nil
	}

	query := buildUnderutilizedPodsQuery(clusterName, namespaceName, tw)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
		Time:  evalTime,
	}
	if !tw.IsZero() {
		timeout := sysdig.Timeout(windowedQueryTimeout)
		params.Timeout = &timeout
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get underutilized pod list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get underutilized pod list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildUnderutilizedPodsQuery(clusterName, namespaceName string, tw TimeWindow) string {
	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf("kube_cluster_name=\"%s\"", clusterName))
	}
	if namespaceName != "" {
		filters = append(filters, fmt.Sprintf("kube_namespace_name=\"%s\"", namespaceName))
	}

	filterString := ""
	if len(filters) > 0 {
		filterString = fmt.Sprintf("{%s}", strings.Join(filters, ","))
	}

	used := fmt.Sprintf("sysdig_container_cpu_cores_used%s", filterString)
	quota := fmt.Sprintf("sysdig_container_cpu_cores_quota_limit%s", filterString)
	if !tw.IsZero() {
		sel := tw.RangeSelector()
		used = fmt.Sprintf("avg_over_time(%s%s)", used, sel)
		quota = fmt.Sprintf("avg_over_time(%s%s)", quota, sel)
	}

	return fmt.Sprintf("sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(%s) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(%s) > 0) < 0.25", used, quota)
}
