package mcp_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	localmcp "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

// dummyTool implements the interface required by Handler.RegisterTools
type dummyTool struct {
	name                string
	requiredPermissions []string
}

func (d *dummyTool) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool(d.name, mcp.WithDescription("dummy tool"))
	// Initialize Meta to avoid nil pointer issues in strict checks
	if tool.Meta == nil {
		tool.Meta = &mcp.Meta{
			AdditionalFields: make(map[string]interface{}),
		}
	}
	if len(d.requiredPermissions) > 0 {
		localmcp.WithRequiredPermissions(d.requiredPermissions...)(&tool)
	}
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("success"), nil
	})
}

var _ = Describe("McpHandler", func() {
	var (
		ctrl       *gomock.Controller
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		handler    *localmcp.Handler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		handler = localmcp.NewHandler(mockClient)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Permissions", func() {
		It("should filter tools based on permissions", func() {
			t1 := &dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}}
			t2 := &dummyTool{name: "tool2", requiredPermissions: []string{"perm2"}}
			t3 := &dummyTool{name: "tool3" /* no permissions required */}

			handler.RegisterTools(t1, t2, t3)

			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &sysdig.UserPermissions{
					Permissions: []string{"perm1"},
				},
			}, nil)

			c := initializeInProcessClient(handler)

			resp, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
			Expect(err).NotTo(HaveOccurred())

			var names []string
			for _, t := range resp.Tools {
				names = append(names, t.Name)
			}
			Expect(names).To(ConsistOf("tool1", "tool3"))
		})

		It("should handle permission errors gracefully", func() {
			t1 := &dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}}
			handler.RegisterTools(t1)

			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(nil, fmt.Errorf("error"))

			c := initializeInProcessClient(handler)

			resp, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Tools).To(BeEmpty())
		})
	})

	Context("HTTP Handlers and Middleware", func() {
		It("AsStreamableHTTP should serve correctly and middleware should extract headers", func(ctx SpecContext) {
			expectedHost := "https://test.sysdig.com"
			expectedToken := "my-token"

			// Register a tool to ensure ListTools triggers checking
			t1 := &dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}}
			handler.RegisterTools(t1)

			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, reqEditors ...sysdig.RequestEditorFn) (*sysdig.GetMyPermissionsResponse, error) {
				req, _ := http.NewRequest("GET", "/", nil)
				authenticator := sysdig.WithHostAndTokenFromContext()
				err := authenticator(c, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(req.Header.Get("Authorization")).To(Equal("Bearer " + expectedToken))
				Expect(req.URL.Scheme).To(Equal("https"))
				Expect(req.URL.Host).To(Equal("test.sysdig.com"))

				return &sysdig.GetMyPermissionsResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200: &sysdig.UserPermissions{
						Permissions: []string{"perm1"},
					},
				}, nil
			})

			// Use root path to simplify MCP routing in test
			h := handler.AsStreamableHTTP("/")
			ts := httptest.NewServer(h)
			defer ts.Close()

			client := ts.Client()
			mcpURL := ts.URL

			testClient := NewHTTPTestClient(client, mcpURL)

			// 1. Initialize
			testClient.Initialize(ctx)

			// 2. List Tools (Triggers permission check)
			headers := map[string]string{
				"X-Sysdig-Host": expectedHost,
				"Authorization": "Bearer " + expectedToken,
			}
			respList := testClient.ListTools(ctx, headers)
			defer func() { _ = respList.Body.Close() }()

			if respList.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(respList.Body)
				fmt.Printf("ListTools failed with status %d: %s\n", respList.StatusCode, string(body))
			}
			Expect(respList.StatusCode).To(Equal(http.StatusOK))
		}, NodeTimeout(time.Second*5))

		It("AsSSE should return a handler", func() {
			h := handler.AsSSE("/sse")
			Expect(h).NotTo(BeNil())
		})

		It("should handle X-Sysdig-Token header in middleware", func(ctx SpecContext) {
			expectedToken := "token-header"

			t1 := &dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}}
			handler.RegisterTools(t1)

			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, reqEditors ...sysdig.RequestEditorFn) (*sysdig.GetMyPermissionsResponse, error) {
				req, _ := http.NewRequest("GET", "/", nil)
				authenticator := sysdig.WithHostAndTokenFromContext()
				err := authenticator(c, req)
				Expect(err).NotTo(HaveOccurred())
				// Host might be missing, which is fine
				Expect(req.Header.Get("Authorization")).To(Equal("Bearer " + expectedToken))

				return &sysdig.GetMyPermissionsResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200: &sysdig.UserPermissions{
						Permissions: []string{"perm1"},
					},
				}, nil
			})

			h := handler.AsStreamableHTTP("/")
			ts := httptest.NewServer(h)
			defer ts.Close()

			client := ts.Client()
			mcpURL := ts.URL

			testClient := NewHTTPTestClient(client, mcpURL)

			// 1. Initialize
			testClient.Initialize(ctx)

			// 2. List Tools
			headers := map[string]string{
				"X-Sysdig-Token": expectedToken,
			}
			respList := testClient.ListTools(ctx, headers)
			defer func() { _ = respList.Body.Close() }()

			Expect(respList.StatusCode).To(Equal(http.StatusOK))
		}, NodeTimeout(time.Second*5))

		It("should handle request with no auth headers", func(ctx SpecContext) {
			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, reqEditors ...sysdig.RequestEditorFn) (*sysdig.GetMyPermissionsResponse, error) {
				return nil, fmt.Errorf("no auth")
			})

			h := handler.AsStreamableHTTP("/")
			ts := httptest.NewServer(h)
			defer ts.Close()

			client := ts.Client()
			mcpURL := ts.URL

			testClient := NewHTTPTestClient(client, mcpURL)

			// 1. Initialize
			testClient.Initialize(ctx)

			// 2. List Tools
			respList := testClient.ListTools(ctx, nil)
			defer func() { _ = respList.Body.Close() }()

			// Should return 200 even if internal logic returned error for tool list (JSON-RPC error inside body)
			Expect(respList.StatusCode).To(Equal(http.StatusOK))
		}, NodeTimeout(time.Second*5))
	})

	Context("Stdio", func() {
		It("ServeStdio should return when context is cancelled", func(ctx SpecContext) {
			c, cancel := context.WithCancel(ctx)
			cancel()

			r, w := io.Pipe()
			defer func() { _ = w.Close() }()

			err := handler.ServeStdio(c, r, io.Discard)
			if err != nil {
				Expect(err).To(HaveOccurred())
			}
		}, NodeTimeout(time.Second*5))
	})
})

func initializeInProcessClient(handler *localmcp.Handler) *client.Client {
	c, err := handler.ServeInProcessClient()
	Expect(err).NotTo(HaveOccurred())
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	Expect(err).NotTo(HaveOccurred())
	return c
}

type HTTPTestClient struct {
	client    *http.Client
	baseURL   string
	sessionID string
}

func NewHTTPTestClient(client *http.Client, baseURL string) *HTTPTestClient {
	return &HTTPTestClient{
		client:  client,
		baseURL: baseURL,
	}
}

func (c *HTTPTestClient) Initialize(ctx context.Context) {
	initBody := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	reqInit, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL, strings.NewReader(initBody))
	reqInit.Header.Set("Content-Type", "application/json")

	respInit, err := c.client.Do(reqInit)
	Expect(err).NotTo(HaveOccurred())
	_ = respInit.Body.Close()

	Expect(respInit.StatusCode).To(Equal(http.StatusOK))
	c.sessionID = respInit.Header.Get("Mcp-Session-Id")
}

func (c *HTTPTestClient) ListTools(ctx context.Context, headers map[string]string) *http.Response {
	listBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	reqList, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL, strings.NewReader(listBody))
	reqList.Header.Set("Content-Type", "application/json")
	if c.sessionID != "" {
		reqList.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	for k, v := range headers {
		reqList.Header.Set(k, v)
	}

	respList, err := c.client.Do(reqList)
	Expect(err).NotTo(HaveOccurred())
	return respList
}
