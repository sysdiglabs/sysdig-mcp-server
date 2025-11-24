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

var _ = Describe("TroubleshootKubernetesListTopRestartedPods Tool", func() {
	var (
		tool       *tools.TroubleshootKubernetesListTopRestartedPods
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewTroubleshootKubernetesListTopRestartedPods(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("troubleshoot_kubernetes_list_top_restarted_pods")).NotTo(BeNil())
	})

	When("listing top restarted pods", func() {
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
				"troubleshoot_kubernetes_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "troubleshoot_kubernetes_list_top_restarted_pods",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total) > 0)`,
				},
			),
			Entry(nil,
				"troubleshoot_kubernetes_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "troubleshoot_kubernetes_list_top_restarted_pods",
						Arguments: map[string]any{"limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total) > 0)`,
				},
			),
			Entry(nil,
				"troubleshoot_kubernetes_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "troubleshoot_kubernetes_list_top_restarted_pods",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total{kube_cluster_name="my_cluster"}) > 0)`,
				},
			),
			Entry(nil,
				"troubleshoot_kubernetes_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "troubleshoot_kubernetes_list_top_restarted_pods",
						Arguments: map[string]any{"namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total{kube_namespace_name="my_namespace"}) > 0)`,
				},
			),
			Entry(nil,
				"troubleshoot_kubernetes_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "troubleshoot_kubernetes_list_top_restarted_pods",
						Arguments: map[string]any{
							"cluster_name":   "my_cluster",
							"namespace_name": "my_namespace",
							"workload_type":  "deployment",
							"workload_name":  "my_workload",
							"pod_name":       "my_pod",
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace",kube_workload_type="deployment",kube_workload_name="my_workload",kube_pod_name="my_pod"}) > 0)`,
				},
			),
		)
	})
})
