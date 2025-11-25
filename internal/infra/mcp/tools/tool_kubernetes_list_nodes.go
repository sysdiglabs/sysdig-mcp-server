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

type KubernetesListNodes struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListNodes(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListNodes {
	return &KubernetesListNodes{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListNodes) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_nodes",
		mcp.WithDescription("Lists the information from all nodes, all nodes from a cluster or a specific node with some name."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithString("node_name", mcp.Description("The name of the node to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of nodes to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions(), // FIXME(fede): Add the required permissions. It should be `promql.exec` but somehow the token does not have that permission even if you are able to execute queries.
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListNodes) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	nodeName := mcp.ParseString(request, "node_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := buildKubeNodeInfoQuery(clusterName, nodeName)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get node list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get node list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}

func buildKubeNodeInfoQuery(clusterName, nodeName string) string {
	filters := []string{}
	if clusterName != "" {
		filters = append(filters, fmt.Sprintf("cluster=\"%s\"", clusterName))
	}
	if nodeName != "" {
		filters = append(filters, fmt.Sprintf("kube_node_name=\"%s\"", nodeName))
	}

	if len(filters) == 0 {
		return "kube_node_info"
	}

	return fmt.Sprintf("kube_node_info{%s}", strings.Join(filters, ","))
}
