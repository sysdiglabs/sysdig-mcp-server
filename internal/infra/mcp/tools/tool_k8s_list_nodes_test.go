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

var _ = Describe("KubernetesListNodes Tool", func() {
	var (
		tool       *tools.K8sListNodes
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewK8sListNodes(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_nodes")).NotTo(BeNil())
	})

	When("listing nodes", func() {
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
				"k8s_list_nodes",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_nodes",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_node_info`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_nodes",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_nodes",
						Arguments: map[string]any{"limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_node_info`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"k8s_list_nodes",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_nodes",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_node_info{cluster="my_cluster"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_nodes",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_nodes",
						Arguments: map[string]any{"node_name": "my_node"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_node_info{kube_node_name="my_node"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_nodes",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_nodes",
						Arguments: map[string]any{"cluster_name": "my_cluster", "node_name": "my_node"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_node_info{cluster="my_cluster",kube_node_name="my_node"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_nodes",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_nodes",
						Arguments: map[string]any{"cluster_name": "my_cluster", "limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_node_info{cluster="my_cluster"}`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
		)
	})
})
