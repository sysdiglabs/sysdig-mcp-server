package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type TroubleshootKubernetesListTopUnavailablePods struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewTroubleshootKubernetesListTopUnavailablePods(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *TroubleshootKubernetesListTopUnavailablePods {
	return &TroubleshootKubernetesListTopUnavailablePods{
		SysdigClient: sysdigClient,
	}
}

func (t *TroubleshootKubernetesListTopUnavailablePods) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("troubleshoot_kubernetes_list_top_unavailable_pods",
		mcp.WithDescription("Shows the top N pods with the highest number of unavailable or unready replicas in a Kubernetes cluster, ordered from highest to lowest."),
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
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *TroubleshootKubernetesListTopUnavailablePods) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	limit := mcp.ParseInt(request, "limit", 20)

	query := buildTopUnavailablePodsQuery(limit, clusterName, namespaceName, workloadType, workloadName)

	params := &sysdig.GetQueryV1Params{
		Query: query,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get top unavailable pods", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get top unavailable pods: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildTopUnavailablePodsQuery(limit int, clusterName, namespaceName, workloadType, workloadName string) string {
	baseFilters := []string{}
	if clusterName != "" {
		baseFilters = append(baseFilters, fmt.Sprintf("kube_cluster_name=\"%s\"", clusterName))
	}
	if namespaceName != "" {
		baseFilters = append(baseFilters, fmt.Sprintf("kube_namespace_name=\"%s\"", namespaceName))
	}
	if workloadType != "" {
		baseFilters = append(baseFilters, fmt.Sprintf("kube_workload_type=\"%s\"", workloadType))
	}
	if workloadName != "" {
		baseFilters = append(baseFilters, fmt.Sprintf("kube_workload_name=\"%s\"", workloadName))
	}

	// Filters for kube_workload_status_desired and kube_daemonset_status_number_ready
	commonFiltersStr := strings.Join(baseFilters, ",")

	// Filters for kube_workload_status_ready (needs extra filter)
	readyFilters := append([]string{"kube_workload_type!=\"daemonset\""}, baseFilters...)
	readyFiltersStr := strings.Join(readyFilters, ",")

	return fmt.Sprintf(`topk (
  %d,
    (
      sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
        kube_workload_status_desired{%s}
      )
    )
  -
      (
          sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
              kube_workload_status_ready{%s}
            or
              kube_daemonset_status_number_ready{%s}
          )
        or
          vector(0)
      )
    >
      0 or vector(0)
)`, limit, commonFiltersStr, readyFiltersStr, commonFiltersStr)
}
