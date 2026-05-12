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

var _ = Describe("KubernetesListTopRestartedPods Tool", func() {
	var (
		tool       *tools.K8sListTopRestartedPods
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
		tool = tools.NewK8sListTopRestartedPods(mockSysdig, mockClock)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_top_restarted_pods")).NotTo(BeNil())
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
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_restarted_pods",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total) > 0)`,
				},
			),
			Entry(nil,
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_restarted_pods",
						Arguments: map[string]any{"limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total) > 0)`,
				},
			),
			Entry(nil,
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_restarted_pods",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total{kube_cluster_name="my_cluster"}) > 0)`,
				},
			),
			Entry(nil,
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_restarted_pods",
						Arguments: map[string]any{"namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (kube_pod_container_status_restarts_total{kube_namespace_name="my_namespace"}) > 0)`,
				},
			),
			Entry(nil,
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_restarted_pods",
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
			Entry("windowed, no filters",
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_restarted_pods",
						Arguments: map[string]any{
							"start": "2026-04-16T10:00:00Z",
							"end":   "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (increase(kube_pod_container_status_restarts_total[3600s])) > 0)`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				),
			),
			Entry("windowed, with filters",
				"k8s_list_top_restarted_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_restarted_pods",
						Arguments: map[string]any{
							"cluster_name": "prod",
							"start":        "2026-04-16T10:00:00Z",
							"end":          "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(10, sum by(pod, kube_cluster_name, kube_namespace_name) (increase(kube_pod_container_status_restarts_total{kube_cluster_name="prod"}[3600s])) > 0)`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				),
			),
		)
	})
})
