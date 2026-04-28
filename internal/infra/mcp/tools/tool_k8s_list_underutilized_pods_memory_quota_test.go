package tools_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	mocks_clock "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock/mocks"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("KubernetesListUnderutilizedPodsMemoryQuota Tool", func() {
	var (
		tool       *tools.K8sListUnderutilizedPodsMemoryQuota
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mockClock  *mocks_clock.MockClock
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		mockClock = mocks_clock.NewMockClock(ctrl)
		mockClock.EXPECT().Now().AnyTimes().Return(time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC))
		tool = tools.NewK8sListUnderutilizedPodsMemoryQuota(mockSysdig, mockClock)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_underutilized_pods_memory_quota")).NotTo(BeNil())
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
				"k8s_list_underutilized_pods_memory_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_underutilized_pods_memory_quota",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_used_bytes) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_limit_bytes) > 0) < 0.25`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_underutilized_pods_memory_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_underutilized_pods_memory_quota",
						Arguments: map[string]any{
							"cluster_name":   "test-cluster",
							"namespace_name": "test-namespace",
							"limit":          20,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_used_bytes{kube_cluster_name="test-cluster",kube_namespace_name="test-namespace"}) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(sysdig_container_memory_limit_bytes{kube_cluster_name="test-cluster",kube_namespace_name="test-namespace"}) > 0) < 0.25`,
					Limit: new(sysdig.LimitQuery(20)),
				},
			),
			Entry("windowed, both start and end",
				"k8s_list_underutilized_pods_memory_quota",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_underutilized_pods_memory_quota",
						Arguments: map[string]any{
							"start": "2026-04-16T10:00:00Z",
							"end":   "2026-04-16T11:00:00Z",
						},
					},
				},
				mergeLimit(newWindowedQueryParams(
					`sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(avg_over_time(sysdig_container_memory_used_bytes[3600s])) / (sum by (kube_cluster_name, kube_namespace_name, kube_pod_name)(avg_over_time(sysdig_container_memory_limit_bytes[3600s])) > 0) < 0.25`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				), 10),
			),
		)
	})
})
