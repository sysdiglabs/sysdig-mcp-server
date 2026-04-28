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

var _ = Describe("KubernetesListTopUnavailablePods Tool", func() {
	var (
		tool       *tools.K8sListTopUnavailablePods
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
		tool = tools.NewK8sListTopUnavailablePods(mockSysdig, mockClock)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("k8s_list_top_unavailable_pods")).NotTo(BeNil())
	})

	When("querying top unavailable pods", func() {
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
				"k8s_list_top_unavailable_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "k8s_list_top_unavailable_pods",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk (
  20,
    (
      sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
        kube_workload_status_desired{}
      )
    )
  -
      (
          sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
              kube_workload_status_ready{kube_workload_type!="daemonset"}
            or
              kube_daemonset_status_number_ready{}
          )
        or
          vector(0)
      )
    >
      0 or vector(0)
)`,
				},
			),
			Entry("with specific limit and cluster",
				"k8s_list_top_unavailable_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_unavailable_pods",
						Arguments: map[string]any{
							"limit":        5,
							"cluster_name": "my-cluster",
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk (
  5,
    (
      sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
        kube_workload_status_desired{kube_cluster_name="my-cluster"}
      )
    )
  -
      (
          sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
              kube_workload_status_ready{kube_workload_type!="daemonset",kube_cluster_name="my-cluster"}
            or
              kube_daemonset_status_number_ready{kube_cluster_name="my-cluster"}
          )
        or
          vector(0)
      )
    >
      0 or vector(0)
)`,
				},
			),
			Entry("with all filters",
				"k8s_list_top_unavailable_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_unavailable_pods",
						Arguments: map[string]any{
							"limit":          10,
							"cluster_name":   "my-cluster",
							"namespace_name": "my-ns",
							"workload_type":  "deployment",
							"workload_name":  "my-app",
						},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `topk (
  10,
    (
      sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
        kube_workload_status_desired{kube_cluster_name="my-cluster",kube_namespace_name="my-ns",kube_workload_type="deployment",kube_workload_name="my-app"}
      )
    )
  -
      (
          sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (
              kube_workload_status_ready{kube_workload_type!="daemonset",kube_cluster_name="my-cluster",kube_namespace_name="my-ns",kube_workload_type="deployment",kube_workload_name="my-app"}
            or
              kube_daemonset_status_number_ready{kube_cluster_name="my-cluster",kube_namespace_name="my-ns",kube_workload_type="deployment",kube_workload_name="my-app"}
          )
        or
          vector(0)
      )
    >
      0 or vector(0)
)`,
				},
			),
			Entry("windowed, no filters (Sysdig-canonical pattern)",
				"k8s_list_top_unavailable_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_unavailable_pods",
						Arguments: map[string]any{
							"start": "2026-04-16T10:00:00Z",
							"end":   "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(20, sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (min_over_time(kube_workload_status_unavailable{}[3600s]) >= 1))`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				),
			),
			Entry("windowed, with cluster filter",
				"k8s_list_top_unavailable_pods",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name: "k8s_list_top_unavailable_pods",
						Arguments: map[string]any{
							"cluster_name": "my-cluster",
							"limit":        5,
							"start":        "2026-04-16T10:00:00Z",
							"end":          "2026-04-16T11:00:00Z",
						},
					},
				},
				newWindowedQueryParams(
					`topk(5, sum by (kube_cluster_name, kube_namespace_name, kube_workload_name) (min_over_time(kube_workload_status_unavailable{kube_cluster_name="my-cluster"}[3600s]) >= 1))`,
					time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC),
				),
			),
		)
	})
})
