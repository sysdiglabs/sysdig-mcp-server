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

var _ = Describe("KubernetesListTopNetworkErrorsInPods Tool", func() {
	var (
		tool       *tools.KubernetesListTopNetworkErrorsInPods
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
		ctx        context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewKubernetesListTopNetworkErrorsInPods(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
		ctx = context.Background()
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("kubernetes_list_top_network_errors_in_pods")).NotTo(BeNil())
	})

	When("listing top network errors", func() {
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
			Entry("default params",
				"kubernetes_list_top_network_errors_in_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_top_network_errors_in_pods",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20,sum(sum_over_time(sysdig_container_net_error_count{}[1h])) by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, kube_pod_name)) / 3600.000000`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry("with custom params",
				"kubernetes_list_top_network_errors_in_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "kubernetes_list_top_network_errors_in_pods",
						Arguments: map[string]any{
							"interval":       "30m",
							"cluster_name":   "prod-cluster",
							"namespace_name": "backend",
							"limit":          5,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(5,sum(sum_over_time(sysdig_container_net_error_count{kube_cluster_name=~"prod-cluster",kube_namespace_name="backend"}[30m])) by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, kube_pod_name)) / 1800.000000`,
					Limit: asPtr(sysdig.LimitQuery(5)),
				},
			),
			Entry("with all params",
				"kubernetes_list_top_network_errors_in_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "kubernetes_list_top_network_errors_in_pods",
						Arguments: map[string]any{
							"interval":       "2h",
							"cluster_name":   "dev",
							"namespace_name": "default",
							"workload_type":  "deployment",
							"workload_name":  "api",
							"limit":          10,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10,sum(sum_over_time(sysdig_container_net_error_count{kube_cluster_name=~"dev",kube_namespace_name="default",kube_workload_type="deployment",kube_workload_name="api"}[2h])) by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, kube_pod_name)) / 7200.000000`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
		)

		It("returns error for invalid interval", func() {
			serverTool := mcpServer.GetTool("kubernetes_list_top_network_errors_in_pods")
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "kubernetes_list_top_network_errors_in_pods",
					Arguments: map[string]any{"interval": "invalid"},
				},
			}
			result, err := serverTool.Handler(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
			Expect(result.Content[0].(mcp.TextContent).Text).To(ContainSubstring("invalid interval format"))
		})

		It("should return an error when sysdig returns an error", func() {
			mockSysdig.EXPECT().GetQueryV1(gomock.Any(), gomock.Any()).Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`{"errors":[{"message":"test error"}]}`)),
			}, nil)

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "kubernetes_list_top_network_errors_in_pods",
					Arguments: map[string]any{},
				},
			}

			serverTool := mcpServer.GetTool("kubernetes_list_top_network_errors_in_pods")
			res, err := serverTool.Handler(ctx, request)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.IsError).To(BeTrue())
			Expect(res.Content[0].(mcp.TextContent).Text).To(ContainSubstring("failed to execute query: status code 500"))
		})
	})
})
