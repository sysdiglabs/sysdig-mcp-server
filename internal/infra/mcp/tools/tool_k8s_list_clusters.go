package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type K8sListClusters struct {
	SysdigClient sysdig.ExtendedClientWithResponsesInterface
	clock        clock.Clock
}

func NewK8sListClusters(sysdigClient sysdig.ExtendedClientWithResponsesInterface, clk clock.Clock) *K8sListClusters {
	return &K8sListClusters{
		SysdigClient: sysdigClient,
		clock:        clk,
	}
}

func (t *K8sListClusters) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool("k8s_list_clusters",
		mcp.WithDescription("Lists the cluster information for all clusters or just the cluster specified. Optionally pass start/end (RFC3339) to list clusters that existed at any point in the window."),
		mcp.WithString("cluster_name", mcp.Description("The name of the cluster to filter by.")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of clusters to return."),
			mcp.DefaultNumber(10),
		),
		WithTimeWindowParams(),
		mcp.WithOutputSchema[map[string]any](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		WithRequiredPermissions("metrics-data.read"),
	)
	s.AddTool(tool, t.handle)
}

func (t *K8sListClusters) handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterName := mcp.ParseString(request, "cluster_name", "")
	limit := mcp.ParseInt(request, "limit", 10)

	tw, err := ParseTimeWindow(request, t.clock)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid time window", err), nil
	}

	query := buildKubeClusterInfoQuery(clusterName, tw)

	limitQuery := sysdig.LimitQuery(limit)
	params := &sysdig.GetQueryV1Params{
		Query: query,
		Limit: &limitQuery,
	}
	if err := tw.ApplyToParams(params); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to build eval time", err), nil
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

func buildKubeClusterInfoQuery(clusterName string, tw TimeWindow) string {
	metric := "kube_cluster_info"
	if clusterName != "" {
		metric = fmt.Sprintf(`kube_cluster_info{cluster="%s"}`, clusterName)
	}
	if !tw.IsZero() {
		return fmt.Sprintf("max_over_time(%s%s) > 0", metric, tw.RangeSelector())
	}
	return metric
}
