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

type TroubleshootKubernetesListTopMemoryConsumedByContainer struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewTroubleshootKubernetesListTopMemoryConsumedByContainer(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *TroubleshootKubernetesListTopMemoryConsumedByContainer {
	return &TroubleshootKubernetesListTopMemoryConsumedByContainer{
		SysdigClient: sysdigClient,
	}
}

func (t *TroubleshootKubernetesListTopMemoryConsumedByContainer) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("troubleshoot_kubernetes_list_top_memory_consumed_by_container",
		mcp.WithDescription("Lists memory-intensive containers."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_type", mcp.Description("The type of the workload to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of containers to return."),
			mcp.DefaultNumber(20),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *TroubleshootKubernetesListTopMemoryConsumedByContainer) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	limit := mcp.ParseInt(request, "limit", 20)

	query := buildTopMemoryConsumedByContainerQuery(clusterName, namespaceName, workloadType, workloadName, limit)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get container list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get container list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildTopMemoryConsumedByContainerQuery(clusterName, namespaceName, workloadType, workloadName string, limit int) string {
	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf(`kube_cluster_name="%s"`, clusterName))
	}
	if namespaceName != "" {
		filters = append(filters, fmt.Sprintf(`kube_namespace_name="%s"`, namespaceName))
	}
	if workloadType != "" {
		filters = append(filters, fmt.Sprintf(`kube_workload_type="%s"`, workloadType))
	}
	if workloadName != "" {
		filters = append(filters, fmt.Sprintf(`kube_workload_name="%s"`, workloadName))
	}

	filterString := ""
	if len(filters) > 0 {
		filterString = "{" + strings.Join(filters, ", ") + "}"
	}

	return fmt.Sprintf(`topk(%d, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, container_label_io_kubernetes_container_name) (sysdig_container_memory_used_bytes%s))`, limit, filterString)
}
