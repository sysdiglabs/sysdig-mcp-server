package tools_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	inframcp "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("ToolRunSysql", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		tool       *tools.ToolRunSysql
		ctrl       *gomock.Controller
		handler    *inframcp.Handler
		mcpClient  *client.Client
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.UserPermissions{
				Permissions: []string{"sage.exec", "risks.read"},
			},
		}, nil).AnyTimes()
		tool = tools.NewToolRunSysql(mockClient)
		handler = inframcp.NewHandler("dev", mockClient)
		handler.RegisterTools(tool)

		var err error
		mcpClient, err = handler.ServeInProcessClient()
		Expect(err).NotTo(HaveOccurred())

		_, err = mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should handle a successful request with a query ending in semicolon", func(ctx SpecContext) {
		sysqlQuery := "MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10;"

		expectedBody := sysdig.QuerySysqlPostJSONRequestBody{
			Q: sysqlQuery,
		}

		mockClient.EXPECT().QuerySysqlPostWithResponse(gomock.Any(), expectedBody).Return(&sysdig.QuerySysqlPostResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.QueryResponse{
				Items: []map[string]any{
					{"id": "vuln-1", "severity": "Critical"},
				},
			},
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_sysql",
				Arguments: map[string]any{
					"sysql_query": sysqlQuery,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())
	})

	It("should add a semicolon if the query does not end with one", func(ctx SpecContext) {
		sysqlQueryWithoutSemicolon := "MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10"
		sysqlQueryWithSemicolon := sysqlQueryWithoutSemicolon + ";"

		// Use DoAndReturn to explicitly verify that the semicolon was added to body.Q
		mockClient.EXPECT().QuerySysqlPostWithResponse(
			gomock.Any(),
			gomock.AssignableToTypeOf(sysdig.QuerySysqlPostJSONRequestBody{}),
		).DoAndReturn(func(ctx context.Context, body sysdig.QuerySysqlPostJSONRequestBody, reqEditors ...sysdig.RequestEditorFn) (*sysdig.QuerySysqlPostResponse, error) {
			// Explicitly verify that the semicolon was added and that it is exactly as expected
			Expect(body.Q).To(Equal(sysqlQueryWithSemicolon), "Expected the query to have a semicolon appended")
			Expect(body.Q).To(HaveSuffix(";"), "Expected the query to end with a semicolon")

			return &sysdig.QuerySysqlPostResponse{
				HTTPResponse: &http.Response{
					StatusCode: 200,
				},
				JSON200: &sysdig.QueryResponse{
					Items: []map[string]any{
						{"id": "vuln-1", "severity": "Critical"},
					},
				},
			}, nil
		})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_sysql",
				Arguments: map[string]any{
					"sysql_query": sysqlQueryWithoutSemicolon,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())
	})

	It("should handle queries with trailing whitespace before semicolon", func(ctx SpecContext) {
		sysqlQueryWithWhitespace := "MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10   ;"

		expectedBody := sysdig.QuerySysqlPostJSONRequestBody{
			Q: sysqlQueryWithWhitespace,
		}

		mockClient.EXPECT().QuerySysqlPostWithResponse(gomock.Any(), expectedBody).Return(&sysdig.QuerySysqlPostResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.QueryResponse{
				Items: []map[string]any{
					{"id": "vuln-1", "severity": "Critical"},
				},
			},
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_sysql",
				Arguments: map[string]any{
					"sysql_query": sysqlQueryWithWhitespace,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())
	})

	It("should return an error if sysql_query is missing", func(ctx SpecContext) {
		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "run_sysql",
				Arguments: map[string]any{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should return an error if sysql_query is empty", func(ctx SpecContext) {
		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_sysql",
				Arguments: map[string]any{
					"sysql_query": "",
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a client error", func(ctx SpecContext) {
		sysqlQuery := "MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10;"

		expectedBody := sysdig.QuerySysqlPostJSONRequestBody{
			Q: sysqlQuery,
		}

		mockClient.EXPECT().QuerySysqlPostWithResponse(gomock.Any(), expectedBody).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_sysql",
				Arguments: map[string]any{
					"sysql_query": sysqlQuery,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a non-200 status code", func(ctx SpecContext) {
		sysqlQuery := "MATCH Vulnerability WHERE severity = 'Critical' LIMIT 10;"

		expectedBody := sysdig.QuerySysqlPostJSONRequestBody{
			Q: sysqlQuery,
		}

		mockClient.EXPECT().QuerySysqlPostWithResponse(gomock.Any(), expectedBody).Return(&sysdig.QuerySysqlPostResponse{
			HTTPResponse: &http.Response{
				StatusCode: 400,
			},
			Body: []byte("Bad Request: Invalid query syntax"),
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "run_sysql",
				Arguments: map[string]any{
					"sysql_query": sysqlQuery,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})
})
