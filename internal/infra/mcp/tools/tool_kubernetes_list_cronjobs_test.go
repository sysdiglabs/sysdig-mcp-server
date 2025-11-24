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

var _ = Describe("KubernetesListCronjobs Tool", func() {
	var (
		tool       *tools.KubernetesListCronjobs
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewKubernetesListCronjobs(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("kubernetes_list_cronjobs")).NotTo(BeNil())
	})

	When("listing cronjobs", func() {
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
				"kubernetes_list_cronjobs",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_cronjobs",
						Arguments: map[string]any{},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cronjob_info`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"kubernetes_list_cronjobs",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_cronjobs",
						Arguments: map[string]any{"limit": "20"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cronjob_info`,
					Limit: asPtr(sysdig.LimitQuery(20)),
				},
			),
			Entry(nil,
				"kubernetes_list_cronjobs",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_cronjobs",
						Arguments: map[string]any{"cluster_name": "my_cluster"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cronjob_info{kube_cluster_name="my_cluster"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"kubernetes_list_cronjobs",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_cronjobs",
						Arguments: map[string]any{"namespace_name": "my_namespace"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cronjob_info{kube_namespace_name="my_namespace"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"kubernetes_list_cronjobs",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_cronjobs",
						Arguments: map[string]any{"cronjob_name": "my_cronjob"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cronjob_info{kube_cronjob_name="my_cronjob"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
			Entry(nil,
				"kubernetes_list_cronjobs",
				mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      "kubernetes_list_cronjobs",
						Arguments: map[string]any{"cluster_name": "my_cluster", "namespace_name": "my_namespace", "cronjob_name": "my_cronjob"},
					},
				},
				sysdig.GetQueryV1Params{
					Query: `kube_cronjob_info{kube_cluster_name="my_cluster",kube_namespace_name="my_namespace",kube_cronjob_name="my_cronjob"}`,
					Limit: asPtr(sysdig.LimitQuery(10)),
				},
			),
		)
	})
})
