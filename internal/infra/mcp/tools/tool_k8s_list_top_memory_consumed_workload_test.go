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

var _ = Describe("KubernetesListTopMemoryConsumedWorkload Tool", func() {
	var (
		tool       *tools.K8sListTopMemoryConsumedWorkload
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewK8sListTopMemoryConsumedWorkload(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_top_memory_consumed_workload")).NotTo(BeNil())
	})

	When("listing top memory consumed by workload", func() {
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
				"k8s_list_top_memory_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_memory_consumed_workload",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name) (sysdig_container_memory_used_bytes))`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"k8s_list_top_memory_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_memory_consumed_workload",
						Arguments: map[string]any{"limit": "10"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name) (sysdig_container_memory_used_bytes))`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_top_memory_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_memory_consumed_workload",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name) (sysdig_container_memory_used_bytes{kube_cluster_name="my_cluster"}))`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"k8s_list_top_memory_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_memory_consumed_workload",
						Arguments: map[string]any{"cluster_name": "my_cluster", "namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name) (sysdig_container_memory_used_bytes{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace"}))`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"k8s_list_top_memory_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_memory_consumed_workload",
						Arguments: map[string]any{
							"cluster_name":   "my_cluster",
							"namespace_name": "my_namespace",
							"workload_type":  "deployment",
							"workload_name":  "my_workload",
							"limit":          "5",
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(5, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name) (sysdig_container_memory_used_bytes{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace",kube_workload_type="deployment",kube_workload_name="my_workload"}))`,
					Limit: asPtr(sysdig.LimitQuery(5)),
				},
			),
		)
	})
})
