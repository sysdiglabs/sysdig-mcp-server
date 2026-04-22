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

var _ = Describe("KubernetesListTopCPUConsumedWorkload Tool", func() {
	var (
		tool       *tools.K8sListTopCPUConsumedWorkload
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
		tool = tools.NewK8sListTopCPUConsumedWorkload(mockSysdig, mockClock)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_top_cpu_consumed_workload")).NotTo(BeNil())
	})

	When("listing top cpu consumed by workload", func() {
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
			Entry("instant snapshot, no params",
				"k8s_list_top_cpu_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_cpu_consumed_workload",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name)(sysdig_container_cpu_cores_used))`,
				},
			),
			Entry("instant snapshot, cluster+namespace filter",
				"k8s_list_top_cpu_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_cpu_consumed_workload",
						Arguments: map[string]any{
							"cluster_name":   "prod",
							"namespace_name": "default",
							"limit":          10,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(10, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name)(sysdig_container_cpu_cores_used{kube_cluster_name="prod",kube_namespace_name="default"}))`,
				},
			),
			Entry("instant snapshot, all filters",
				"k8s_list_top_cpu_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_cpu_consumed_workload",
						Arguments: map[string]any{
							"cluster_name":   "prod",
							"namespace_name": "default",
							"workload_name":  "api",
							"workload_type":  "deployment",
							"limit":          5,
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk(5, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name)(sysdig_container_cpu_cores_used{kube_cluster_name="prod",kube_namespace_name="default",kube_workload_type="deployment",kube_workload_name="api"}))`,
				},
			),
			Entry("windowed, both start and end",
				"k8s_list_top_cpu_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_cpu_consumed_workload",
						Arguments: map[string]any{
							"start": "2026-04-16T10:00:00Z",
							"end":   "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name)(avg_over_time(sysdig_container_cpu_cores_used[3600s])))`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				),
			),
			Entry("windowed, start only defaults end to now",
				"k8s_list_top_cpu_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_cpu_consumed_workload",
						Arguments: map[string]any{
							"start": "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name)(avg_over_time(sysdig_container_cpu_cores_used[3600s])))`,
					time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC),
				),
			),
			Entry("windowed, with filters",
				"k8s_list_top_cpu_consumed_workload",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_cpu_consumed_workload",
						Arguments: map[string]any{
							"cluster_name": "prod",
							"start":        "2026-04-16T10:00:00Z",
							"end":          "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_type, kube_workload_name)(avg_over_time(sysdig_container_cpu_cores_used{kube_cluster_name="prod"}[3600s])))`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				),
			),
		)
	})

	When("the time window is invalid", func() {
		It("rejects an unparseable start", func() {
			serverTool := mcpServer.GetTool("k8s_list_top_cpu_consumed_workload")
			result, err := serverTool.Handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "k8s_list_top_cpu_consumed_workload",
					Arguments: map[string]any{"start": "not-a-timestamp"},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("rejects an end before start", func() {
			serverTool := mcpServer.GetTool("k8s_list_top_cpu_consumed_workload")
			result, err := serverTool.Handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "k8s_list_top_cpu_consumed_workload",
					Arguments: map[string]any{
						"start": "2026-04-16T11:00:00Z",
						"end":   "2026-04-16T10:00:00Z",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})
})
