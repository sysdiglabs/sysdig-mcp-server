package mcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("ToolGenerateSysql", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		tool       *ToolGenerateSysql
		ctrl       *gomock.Controller
		handler    *Handler
		mcpClient  *client.Client
		checker    PermissionChecker
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		checker = NewPermissionChecker(mockClient)
		mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.UserPermissions{
				Permissions: []string{"sage.exec"},
			},
		}, nil)
		tool = NewToolGenerateSysql(mockClient, checker)
		handler = NewHandlerWithTools(tool)

		var err error
		mcpClient, err = handler.ServeInProcessClient()
		Expect(err).NotTo(HaveOccurred())

		_, err = mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should handle a successful request", func(ctx SpecContext) {
		question := "all vulnerabilities across my workloads"
		mockClient.EXPECT().GenerateSysqlWithResponse(gomock.Any(), question).Return(&sysdig.GenerateSysqlResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.SysqlQuery{
				Text: "MATCH KubeWorkload AFFECTED_BY Vulnerability RETURN KubeWorkload, Vulnerability;\n",
			},
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "generate_sysql",
				Arguments: map[string]interface{}{
					"question": question,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.StructuredContent).To(Equal(map[string]interface{}{"text": "MATCH KubeWorkload AFFECTED_BY Vulnerability RETURN KubeWorkload, Vulnerability;\n"}))
		Expect(result.IsError).To(BeFalse())
	})

	It("should return an error if question is missing", func(ctx SpecContext) {
		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "generate_sysql",
				Arguments: map[string]interface{}{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
		Expect(result.Content[0]).To(Equal(mcp.TextContent{Type: "text", Text: "question is required"}))
	})

	It("should handle a client error", func(ctx SpecContext) {
		question := "what is bash"
		mockClient.EXPECT().GenerateSysqlWithResponse(gomock.Any(), question).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "generate_sysql",
				Arguments: map[string]interface{}{
					"question": question,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
		Expect(result.Content[0]).To(Equal(mcp.TextContent{Type: "text", Text: "error triggering request: client error"}))
	})

	It("should handle a non-200 status code", func(ctx SpecContext) {
		question := "what is bash"
		mockClient.EXPECT().GenerateSysqlWithResponse(gomock.Any(), question).Return(&sysdig.GenerateSysqlResponse{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			Body: []byte("Not Found"),
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "generate_sysql",
				Arguments: map[string]interface{}{
					"question": question,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
		Expect(result.Content[0]).To(Equal(mcp.TextContent{Type: "text", Text: "error generating SysQL query, status code: 404, response: Not Found"}))
	})
})
