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

type KubernetesListWorkloads struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListWorkloads(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListWorkloads {
	return &KubernetesListWorkloads{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListWorkloads) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_workloads",
		mcp.WithDescription("Lists all the workloads that are in a particular state, desired, ready, running or unavailable. The LLM can filter by cluster, namespace, workload name or type."),
		mcp.WithString("status",
			mcp.Description("The status of the workload."),
			mcp.Enum("desired", "ready", "running", "unavailable"),
			mcp.Required(),
		),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("workload_name", mcp.Description("The name of the workload to filter by.")),
		mcp.WithString("workload_type",
			mcp.Description("The type of the workload."),
			mcp.Enum("deployment", "daemonset", "statefulset"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of workloads to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListWorkloads) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := mcp.ParseString(request, "status", "")
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	workloadName := mcp.ParseString(request, "workload_name", "")
	workloadType := mcp.ParseString(request, "workload_type", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := buildKubeWorkloadInfoQuery(status, clusterName, namespaceName, workloadName, workloadType)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
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

func buildKubeWorkloadInfoQuery(status, clusterName, namespaceName, workloadName, workloadType string) string {
	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf("kube_cluster_name=\"%s\"", clusterName))
	}
	if namespaceName != "" {
		filters = append(filters, fmt.Sprintf("kube_namespace_name=\"%s\"", namespaceName))
	}
	if workloadName != "" {
		filters = append(filters, fmt.Sprintf("kube_workload_name=\"%s\"", workloadName))
	}
	if workloadType != "" {
		filters = append(filters, fmt.Sprintf("kube_workload_type=\"%s\"", workloadType))
	}

	metric := fmt.Sprintf("kube_workload_status_%s", status)

	if len(filters) == 0 {
		return metric
	}

	return fmt.Sprintf("%s{%s}", metric, strings.Join(filters, ","))
}
