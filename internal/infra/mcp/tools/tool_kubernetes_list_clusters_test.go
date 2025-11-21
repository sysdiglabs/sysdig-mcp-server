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

var _ = Describe("KubernetesListClusters Tool", func() {
	var (
		tool       *tools.KubernetesListClusters
		mockSysdig *mocks.MockExtendedClientWithResponsesInterface
		mcpServer  *server.MCPServer
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockSysdig = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		tool = tools.NewKubernetesListClusters(mockSysdig)
		mcpServer = server.NewMCPServer("test", "test")
		tool.RegisterInServer(mcpServer)
	})

	It("should register successfully in the server", func() {
		Expect(mcpServer.GetTool("kubernetes_list_clusters")).NotTo(BeNil())
	})

	When("listing all clusters", func() {
		It("succeeds", func() {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "kubernetes_list_clusters",
					Arguments: map[string]any{},
				},
			}

			query := "kube_cluster_info"
			limit := sysdig.LimitQuery(10)
			params := &sysdig.GetQueryV1Params{
				Query: query,
				Limit: &limit,
			}

			apiResponse := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"success"}`)),
			}

			mockSysdig.EXPECT().GetQueryV1(gomock.Any(), params).Return(apiResponse, nil)

			serverTool := mcpServer.GetTool("kubernetes_list_clusters")
			result, err := serverTool.Handler(context.Background(), request)
			Expect(err).NotTo(HaveOccurred())

			resultData, ok := result.StructuredContent.(sysdig.QueryResponseV1)
			Expect(ok).To(BeTrue())
			status := sysdig.QueryResponseV1StatusSuccess
			Expect(resultData.Status).To(Equal(&status))
		})
	})
})
