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

var _ = Describe("KubernetesListUnderutilizedPodsCPUQuota Tool", func() {
	var (
		tool       *tools.K8sListUnderutilizedPodsCPUQuota
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewK8sListUnderutilizedPodsCPUQuota(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_underutilized_pods_cpu_quota")).NotTo(BeNil())
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
				"k8s_list_underutilized_pods_cpu_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_underutilized_pods_cpu_quota",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_used) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_quota_limit) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_underutilized_pods_cpu_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_underutilized_pods_cpu_quota",
						Arguments: map[string]any{"limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_used) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_quota_limit) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"k8s_list_underutilized_pods_cpu_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_underutilized_pods_cpu_quota",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_used{kube_cluster_name="my_cluster"}) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_quota_limit{kube_cluster_name="my_cluster"}) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_underutilized_pods_cpu_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_underutilized_pods_cpu_quota",
						Arguments: map[string]any{"namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_used{kube_namespace_name="my_namespace"}) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_quota_limit{kube_namespace_name="my_namespace"}) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_underutilized_pods_cpu_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_underutilized_pods_cpu_quota",
						Arguments: map[string]any{"cluster_name": "my_cluster", "namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_used{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace"}) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_cpu_cores_quota_limit{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace"}) > 0) < 0.25`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
		)
	})
})
