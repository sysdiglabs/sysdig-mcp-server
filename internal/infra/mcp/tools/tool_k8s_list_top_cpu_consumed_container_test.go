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

var _ = Describe("KubernetesListTopCPUConsumedContainer Tool", func() {
	var (
		tool       *tools.K8sListTopCPUConsumedContainer
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewK8sListTopCPUConsumedContainer(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_top_cpu_consumed_container")).NotTo(BeNil())
	})

	When("listing top cpu consumed by container", func() {
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
			Expect(resultData.Text).To(ContainSubstring(`"status":"success"`))
		},
			Entry("with no params", context.Background(), "k8s_list_top_cpu_consumed_container",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_cpu_consumed_container",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, container_label_io_kubernetes_container_name)(sysdig_container_cpu_cores_used))`,
				},
			),
			Entry("with all params", context.Background(), "k8s_list_top_cpu_consumed_container",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_cpu_consumed_container",
						Arguments: map[string]any{
							"cluster_name":   "test-cluster",
							"namespace_name": "test-namespace",
							"workload_type":  "deployment",
							"workload_name":  "test-workload",
							"limit":          10,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name, container_label_io_kubernetes_container_name)(sysdig_container_cpu_cores_used{kube_cluster_name="test-cluster",kube_namespace_name="test-namespace",kube_workload_type="deployment",kube_workload_name="test-workload"}))`,
				},
			),
		)
	})
})
