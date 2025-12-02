package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type KubernetesListClusters struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
}

func NewKubernetesListClusters(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *KubernetesListClusters {
	return &KubernetesListClusters{
		SysdigClient: sysdigClient,
	}
}

func (t *KubernetesListClusters) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("kubernetes_list_clusters",
		mcp.WithDescription("Lists the cluster information for all clusters or just the cluster specified."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of clusters to return."),
			mcp.DefaultNumber(10),
		),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *KubernetesListClusters) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	query := "kube_cluster_info"
	if clusterName != "" {
		query = fmt.Sprintf("kube_cluster_info{cluster=\"%s\"}", clusterName)
	}

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}

	httpResp, err := t.SysdigClient.GetQueryV1(ctx, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get cluster list", err), nil
	}

	if httpResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return mcp.NewToolResultErrorf("failed to get cluster list: status code %d, body: %s", httpResp.StatusCode, string(bodyBytes)), nil
	}

	var queryResponse sysdig.QueryResponseV1
	if err := json.NewDecoder(httpResp.Body).Decode(&queryResponse); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to decode response", err), nil
	}

	return mcp.NewToolResultJSON(queryResponse)
}
