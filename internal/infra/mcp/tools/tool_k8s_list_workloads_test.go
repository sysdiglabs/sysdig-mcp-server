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

var _ = Describe("KubernetesListWorkloads Tool", func() {
	var (
		tool       *tools.K8sListWorkloads
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
		tool = tools.NewK8sListWorkloads(mockSysdig, mockClock)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_workloads")).NotTo(BeNil())
	})

	When("listing workloads", func() {
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
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_workloads",
						Arguments: map[string]any{"status": "desired"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_desired`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_workloads",
						Arguments: map[string]any{"status": "ready", "limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_ready`,
					Limit: new(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_workloads",
						Arguments: map[string]any{"status": "running", "cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_running{kube_cluster_name="my_cluster"}`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_workloads",
						Arguments: map[string]any{"status": "unavailable", "namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_unavailable{kube_namespace_name="my_namespace"}`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_workloads",
						Arguments: map[string]any{"status": "desired", "workload_name": "my_workload"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_desired{kube_workload_name="my_workload"}`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_workloads",
						Arguments: map[string]any{"status": "ready", "workload_type": "deployment"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_ready{kube_workload_type="deployment"}`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_workloads",
						Arguments: map[string]any{
							"status":         "running",
							"cluster_name":   "my_cluster",
							"namespace_name": "my_namespace",
							"workload_name":  "my_workload",
							"workload_type":  "statefulset",
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_workload_status_running{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace",kube_workload_name="my_workload",kube_workload_type="statefulset"}`,
					Limit: new(sysdig.LimitQuery(10)),
				},
			),
			Entry("windowed, ready status (no > 0 guard)",
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_workloads",
						Arguments: map[string]any{
							"status": "ready",
							"start":  "2026-04-16T10:00:00Z",
							"end":    "2026-04-16T11:00:00Z",
						},
					},
				},
				mergeLimit(newWindowedQueryParams(
					`max_over_time(kube_workload_status_ready[3600s])`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				), 10),
			),
			Entry("windowed, desired status (no > 0 guard)",
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_workloads",
						Arguments: map[string]any{
							"status":       "desired",
							"cluster_name": "prod",
							"start":        "2026-04-16T10:00:00Z",
							"end":          "2026-04-16T11:00:00Z",
						},
					},
				},
				mergeLimit(newWindowedQueryParams(
					`max_over_time(kube_workload_status_desired{kube_cluster_name="prod"}[3600s])`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				), 10),
			),
			Entry("windowed, unavailable status (> 0 guard)",
				"k8s_list_workloads",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_workloads",
						Arguments: map[string]any{
							"status": "unavailable",
							"start":  "2026-04-16T10:00:00Z",
							"end":    "2026-04-16T11:00:00Z",
						},
					},
				},
				mergeLimit(newWindowedQueryParams(
					`max_over_time(kube_workload_status_unavailable[3600s]) > 0`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				), 10),
			),
		)
	})
})
