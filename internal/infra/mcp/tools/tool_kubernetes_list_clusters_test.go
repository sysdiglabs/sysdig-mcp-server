package tools_test

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
	"go.uber.org/mock/gomock"
)

var _ = Describe("KubernetesListClusters Tool", func() {
	var (
		tool       *tools.KubernetesListClusters
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewKubernetesListClusters(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("kubernetes_list_clusters")).NotTo(BeNil())
	})

	When("listing all clusters", func() {
		DescribeTable("it succeeds", func(ctx context.Context, toolName string, request mcp.CallToolRequest, expectedParamsRequested sysdig.GetQueryV1Params) {
			mockSysdig.EXPECT().GetQueryV1(gomock.Any(), &expectedParamsRequested).Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"success"}`)),
			}, nil)

			serverTool := mcpServer.GetTool(toolName)
			result, err := serverTool.Handler(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			resultData, ok := result.Content[0].(mcp.TextContent)
			Expect(ok).To(BeTrue())
			Expect(resultData.Text).To(MatchJSON(`{"status":"success"}`))
		},
			Entry(nil,
				"kubernetes_list_clusters",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_clusters",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cluster_info`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"kubernetes_list_clusters",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_clusters",
						Arguments: map[string]any{"limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cluster_info`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"kubernetes_list_clusters",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_clusters",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cluster_info{cluster="my_cluster"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"kubernetes_list_clusters",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_clusters",
						Arguments: map[string]any{"cluster_name": "my_cluster", "limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cluster_info{cluster="my_cluster"}`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
		)
	})
})

func asPtr[T any](arg T) *T {
	return &arg
}
