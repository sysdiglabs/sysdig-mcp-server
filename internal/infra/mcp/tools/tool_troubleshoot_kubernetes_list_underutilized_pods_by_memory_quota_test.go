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

var _ = Describe("TroubleshootKubernetesListUnderutilizedPodsByMemoryQuota Tool", func() {
	var (
		tool       *tools.TroubleshootKubernetesListUnderutilizedPodsByMemoryQuota
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewTroubleshootKubernetesListUnderutilizedPodsByMemoryQuota(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota")).NotTo(BeNil())
	})

	When("listing underutilized pods", func() {
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
				"troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_used_bytes) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_limit_bytes) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "troubleshoot_kubernetes_list_underutilized_pods_by_memory_quota",
						Arguments: map[string]any{
							"cluster_name":   "test-cluster",
							"namespace_name": "test-namespace",
							"limit":          20,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_used_bytes{kube_cluster_name="test-cluster",kube_namespace_name="test-namespace"}) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_limit_bytes{kube_cluster_name="test-cluster",kube_namespace_name="test-namespace"}) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
		)
	})
})
