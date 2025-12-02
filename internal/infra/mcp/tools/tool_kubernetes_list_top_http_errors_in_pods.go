package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type KubernetesListTopHttpErrorsInPods struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListTopHttpErrorsInPods(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListTopHttpErrorsInPods {
	return &KubernetesListTopHttpErrorsInPods{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListTopHttpErrorsInPods) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_top_http_errors_in_pods",
		mcp.WithDescription("Lists the pods with the highest rate of HTTP 4xx and 5xx errors over a specified time interval, allowing filtering by cluster, namespace, workload type, and workload name."),
		mcp.WithString("interval", mcp.Description("Time interval for the query (e.g. '1h', '30m'). Default is '1h'.")),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_type", mcp.Description("The type of the workload to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pods to return."),
			mcp.DefaultNumber(20),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListTopHttpErrorsInPods) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	interval := mcp.ParseString(request, "interval", "1h")
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	limit := mcp.ParseInt(request, "limit", 20)

	query, err := buildTopHttpErrorsQuery(interval, limit, clusterName, namespaceName, workloadType, workloadName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to build query", err), nil
	}

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to execute query", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to execute query: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildTopHttpErrorsQuery(interval string, limit int, clusterName, namespaceName, workloadType, workloadName string) (string, error) {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		return "", fmt.Errorf("invalid interval format: %w", err)
	}
	seconds := duration.Seconds()

	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf("kube_cluster_name=~\"%s\"", clusterName))
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

	filterStr := ""
	if len(filters) > 0 {
		filterStr = strings.Join(filters, ",")
	}

	// topk(20,sum(sum_over_time(sysdig_container_net_http_error_count{kube_cluster_name=~"demo-kube-gke"}[1h])) by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, kube_pod_name)) / 3600
	return fmt.Sprintf("topk(%d,sum(sum_over_time(sysdig_container_net_http_error_count{%s}[%s])) by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, kube_pod_name)) / %f",
		limit, filterStr, interval, seconds), nil
}
