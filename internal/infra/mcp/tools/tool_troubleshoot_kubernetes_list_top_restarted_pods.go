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

type TroubleshootKubernetesListTopRestartedPods struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewTroubleshootKubernetesListTopRestartedPods(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *TroubleshootKubernetesListTopRestartedPods {
	return &TroubleshootKubernetesListTopRestartedPods{
		SysdigClient: sysdigClient,
	}
}

func (t *TroubleshootKubernetesListTopRestartedPods) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("troubleshoot_kubernetes_list_top_restarted_pods",
		mcp.WithDescription("Lists the pods with the highest number of container restarts in the specified scope (cluster, namespace, workload, or individual pod). By default, it returns the top 10."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_type", mcp.Description("The type of the workload to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithString("pod_name", mcp.Description("The name of the pod to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pods to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *TroubleshootKubernetesListTopRestartedPods) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	podName := mcp.ParseString(request, "pod_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := buildKubeTopRestartsQuery(clusterName, namespaceName, workloadType, workloadName, podName, limit)

	params := &sysdig.GetQueryV1Params{
		Query: query,
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

func buildKubeTopRestartsQuery(clusterName, namespaceName, workloadType, workloadName, podName string, limit int) string {
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

	return fmt.Sprintf("topk(%d, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total%s) > 0)", limit, filterString)
}
