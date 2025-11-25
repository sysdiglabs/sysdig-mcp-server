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

type TroubleshootKubernetesListCountPodsPerCluster struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewTroubleshootKubernetesListCountPodsPerCluster(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *TroubleshootKubernetesListCountPodsPerCluster {
	return &TroubleshootKubernetesListCountPodsPerCluster{
		SysdigClient: sysdigClient,
	}
}

func (t *TroubleshootKubernetesListCountPodsPerCluster) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("troubleshoot_kubernetes_list_count_pods_per_cluster",
		mcp.WithDescription("List the count of running Kubernetes Pods grouped by cluster and namespace."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return."),
			mcp.DefaultNumber(20),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *TroubleshootKubernetesListCountPodsPerCluster) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	limit := mcp.ParseInt(request, "limit", 20)

	query := buildKubePodCountQuery(clusterName, namespaceName)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get pod count", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get pod count: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildKubePodCountQuery(clusterName, namespaceName string) string {
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

	return fmt.Sprintf("sum by (kube_cluster_name, kube_namespace_name) (kube_pod_info%s)", filterString)
}
