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

type KubernetesListUnderutilizedPodsCPUQuota struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListUnderutilizedPodsCPUQuota(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListUnderutilizedPodsCPUQuota {
	return &KubernetesListUnderutilizedPodsCPUQuota{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListUnderutilizedPodsCPUQuota) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_underutilized_pods_cpu_quota",
		mcp.WithDescription("List Kubernetes pods with CPU usage below 25% of the quota limit."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pods to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListUnderutilizedPodsCPUQuota) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := buildUnderutilizedPodsQuery(clusterName, namespaceName)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
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

func buildUnderutilizedPodsQuery(clusterName, namespaceName string) string {
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

	return fmt.Sprintf("sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_used%s) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_quota_limit%s) > 0) < 0.25", filterString, filterString)
}
