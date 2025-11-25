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

type KubernetesListCronjobs struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListCronjobs(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListCronjobs {
	return &KubernetesListCronjobs{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListCronjobs) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_cronjobs",
		mcp.WithDescription("Retrieves information from the cronjobs in the cluster."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("namespace_name", mcp.Description("The name of the namespace to filter by.")),
		mcp.WithString("cronjob_name", mcp.Description("The name of the cronjob to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of cronjobs to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListCronjobs) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	namespaceName := mcp.ParseString(request, "namespace_name", "")
	cronjobName := mcp.ParseString(request, "cronjob_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := buildKubeCronjobInfoQuery(clusterName, namespaceName, cronjobName)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get cronjob list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get cronjob list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildKubeCronjobInfoQuery(clusterName, namespaceName, cronjobName string) string {
	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf("kube_cluster_name=\"%s\"", clusterName))
	}
	if namespaceName != "" {
		filters = append(filters, fmt.Sprintf("kube_namespace_name=\"%s\"", namespaceName))
	}
	if cronjobName != "" {
		filters = append(filters, fmt.Sprintf("kube_cronjob_name=\"%s\"", cronjobName))
	}

	if len(filters) == 0 {
		return "kube_cronjob_info"
	}

	return fmt.Sprintf("kube_cronjob_info{%s}", strings.Join(filters, ","))
}
