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

type KubernetesListPodContainers struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListPodContainers(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListPodContainers {
	return &KubernetesListPodContainers{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListPodContainers) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_pod_containers",
		mcp.WithDescription("Retrieves information from a particular pod and container."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_type", mcp.Description("The type of the workload to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithString("pod_name", mcp.Description("The name of the pod to filter by.")),
		mcp.WithString("container_name", mcp.Description("The name of the container to filter by.")),
		mcp.WithString("image_pullstring", mcp.Description("The image pullstring to filter by.")),
		mcp.WithString("node_name", mcp.Description("The name of the node to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pod containers to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListPodContainers) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	podName := mcp.ParseString(request, "pod_name", "")
	containerName := mcp.ParseString(request, "container_name", "")
	imagePullstring := mcp.ParseString(request, "image_pullstring", "")
	nodeName := mcp.ParseString(request, "node_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := buildKubePodContainerInfoQuery(clusterName, namespaceName, workloadType, workloadName, podName, containerName, imagePullstring, nodeName)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get pod container list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get pod container list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildKubePodContainerInfoQuery(clusterName, namespaceName, workloadType, workloadName, podName, containerName, imagePullstring, nodeName string) string {
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
	if containerName != "" {
		filters = append(filters, fmt.Sprintf("kube_pod_container_name=\"%s\"", containerName))
	}
	if imagePullstring != "" {
		filters = append(filters, fmt.Sprintf("image=\"%s\"", imagePullstring))
	}
	if nodeName != "" {
		filters = append(filters, fmt.Sprintf("kube_node_name=\"%s\"", nodeName))
	}

	if len(filters) == 0 {
		return "kube_pod_container_info"
	}

	return fmt.Sprintf("kube_pod_container_info{%s}", strings.Join(filters, ","))
}
