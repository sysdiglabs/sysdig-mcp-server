package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
			AdditionalFields: make(map[string]any),
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
		slog.SetDefault(slog.New(slog.NewTextHandler(GinkgoWriter, &slog.HandlerOptions{Level: slog.LevelDebug})))
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
		var testClient *HTTPTestClient

		BeforeEach(func() {
			// Default middleware setup for HTTP tests
			h := handler.AsStreamableHTTP("/")
			testClient = NewHTTPTestClient(h)
		})

		It("AsStreamableHTTP should serve correctly and middleware should extract headers", func(ctx SpecContext) {
			expectedHost := "https://test.sysdig.com"
			expectedToken := "my-token"

			t1 := &dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}}
			handler.RegisterTools(t1)

			mockClient.EXPECT().
				GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).
				DoAndReturn(func(c context.Context, reqEditors ...sysdig.RequestEditorFn) (*sysdig.GetMyPermissionsResponse, error) {
					Expect(sysdig.GetHostFromContext(c)).To(Equal(expectedHost))
					Expect(sysdig.GetTokenFromContext(c)).To(Equal(expectedToken))

					return &sysdig.GetMyPermissionsResponse{
						HTTPResponse: &http.Response{StatusCode: 200},
						JSON200: &sysdig.UserPermissions{
							Permissions: []string{"perm1"},
						},
					}, nil
				})

			testClient.Initialize(ctx)

			headers := map[string]string{
				"X-Sysdig-Host": expectedHost,
				"Authorization": "Bearer " + expectedToken,
			}
			respList := testClient.ListTools(ctx, headers)
			Expect(respList.StatusCode).To(Equal(http.StatusOK))
		}, NodeTimeout(time.Second*5))

		It("should handle X-Sysdig-Token header in middleware", func(ctx SpecContext) {
			expectedToken := "token-header"

			handler.RegisterTools(&dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}})

			mockClient.EXPECT().
				GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).
				DoAndReturn(func(c context.Context, reqEditors ...sysdig.RequestEditorFn) (*sysdig.GetMyPermissionsResponse, error) {
					Expect(sysdig.GetTokenFromContext(c)).To(Equal(expectedToken))

					return &sysdig.GetMyPermissionsResponse{
						HTTPResponse: &http.Response{StatusCode: 200},
						JSON200:      &sysdig.UserPermissions{Permissions: []string{"perm1"}},
					}, nil
				})

			testClient.Initialize(ctx)

			headers := map[string]string{"X-Sysdig-Token": expectedToken}
			respList := testClient.ListTools(ctx, headers)
			Expect(respList.StatusCode).To(Equal(http.StatusOK))
		}, NodeTimeout(time.Second*5))

		It("should handle request with no auth headers", func(ctx SpecContext) {
			mockClient.EXPECT().
				GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("no auth"))

			testClient.Initialize(ctx)
			respList := testClient.ListTools(ctx, nil)
			Expect(respList.StatusCode).To(Equal(http.StatusOK))
		}, NodeTimeout(time.Second*5))

		It("AsSSE should return a handler", func() {
			h := handler.AsSSE("/sse")
			Expect(h).NotTo(BeNil())
		})
	})

	Context("Stdio", func() {
		It("ServeStdio should return when context is cancelled", func(ctx SpecContext) {
			c, cancel := context.WithCancel(ctx)
			cancel()

			r, w := io.Pipe()
			defer func() { _ = w.Close() }()

			err := handler.ServeStdio(c, r, io.Discard)
			// ServeStdio typically returns when the context is canceled or IO stream ends.
			// We just want to ensure it doesn't block indefinitely and exits.
			if err != nil {
				Expect(err).To(HaveOccurred())
			}
		}, NodeTimeout(time.Second*5))
	})
})

// Helpers

func initializeInProcessClient(handler *localmcp.Handler) *client.Client {
	c, err := handler.ServeInProcessClient()
	Expect(err).NotTo(HaveOccurred())
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{})
	Expect(err).NotTo(HaveOccurred())
	return c
}

type HTTPTestClient struct {
	handler   http.Handler
	sessionID string
}

func NewHTTPTestClient(handler http.Handler) *HTTPTestClient {
	return &HTTPTestClient{
		handler: handler,
	}
}

// RPC sends a JSON-RPC 2.0 request
func (c *HTTPTestClient) RPC(ctx context.Context, method string, params any, headers map[string]string) *http.Response {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}

	body, err := json.Marshal(payload)
	Expect(err).NotTo(HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, "POST", "/", bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, req)
	return recorder.Result()
}

func (c *HTTPTestClient) Initialize(ctx context.Context) {
	params := mcp.InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: mcp.Implementation{
			Name:    "test",
			Version: "1.0",
		},
		Capabilities: mcp.ClientCapabilities{},
	}

	resp := c.RPC(ctx, "initialize", params, nil)
	defer func() { _ = resp.Body.Close() }()

	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	c.sessionID = resp.Header.Get("Mcp-Session-Id")
}

func (c *HTTPTestClient) ListTools(ctx context.Context, headers map[string]string) *http.Response {
	return c.RPC(ctx, "tools/list", nil, headers)
}
