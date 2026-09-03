package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	infraauth "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/auth"
	localmcp "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

const (
	validMCPToken = "mcp-access-token"
	allowedOrigin = "https://client.example.com"
	resourceURL   = "https://mcp.example.com/sysdig-mcp-server"
)

type dummyTool struct {
	name                string
	requiredPermissions []string
}

func (d *dummyTool) RegisterInServer(s *server.MCPServer) {
	tool := mcp.NewTool(d.name, mcp.WithDescription("dummy tool"))
	if tool.Meta == nil {
		tool.Meta = &mcp.Meta{AdditionalFields: make(map[string]any)}
	}
	if len(d.requiredPermissions) > 0 {
		tools.WithRequiredPermissions(d.requiredPermissions...)(&tool)
	}
	s.AddTool(tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("success"), nil
	})
}

type fakeTokenVerifier struct {
	validToken string
	calls      int
}

func (v *fakeTokenVerifier) Verify(_ context.Context, rawToken string) error {
	v.calls++
	if rawToken == "insufficient-scope" {
		return infraauth.ErrInsufficientScope
	}
	if rawToken != v.validToken {
		return errors.New("invalid access token")
	}
	return nil
}

func remoteSecurity(verifier *fakeTokenVerifier) localmcp.RemoteSecurity {
	return localmcp.NewRemoteSecurity(
		verifier,
		resourceURL,
		"https://identity.example.com",
		[]string{"mcp:tools"},
		[]string{allowedOrigin},
	)
}

func authorizationHeaders() http.Header {
	return http.Header{"Authorization": []string{"Bearer " + validMCPToken}}
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
		handler = localmcp.NewHandler("dev", mockClient)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Permissions", func() {
		It("filters tools based on permissions", func() {
			handler.RegisterTools(
				&dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}},
				&dummyTool{name: "tool2", requiredPermissions: []string{"perm2"}},
				&dummyTool{name: "tool3"},
			)

			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &sysdig.UserPermissions{Permissions: []string{"perm1"}},
			}, nil)

			c := initializeInProcessClient(handler)
			resp, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
			Expect(err).NotTo(HaveOccurred())

			var names []string
			for _, tool := range resp.Tools {
				names = append(names, tool.Name)
			}
			Expect(names).To(ConsistOf("tool1", "tool3"))
		})

		It("handles permission errors without exposing tools", func() {
			handler.RegisterTools(&dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}})
			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(nil, fmt.Errorf("permission lookup failed"))

			c := initializeInProcessClient(handler)
			resp, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Tools).To(BeEmpty())
		})
	})

	Context("Remote trust boundary", func() {
		var (
			verifier   *fakeTokenVerifier
			testClient *HTTPTestClient
		)

		BeforeEach(func() {
			verifier = &fakeTokenVerifier{validToken: validMCPToken}
			testClient = NewHTTPTestClient(handler.AsStreamableHTTP("/", false, remoteSecurity(verifier)))
		})

		It("serves an authenticated Streamable HTTP session", func(ctx SpecContext) {
			handler.RegisterTools(&dummyTool{name: "tool1", requiredPermissions: []string{"perm1"}})
			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &sysdig.UserPermissions{Permissions: []string{"perm1"}},
			}, nil)

			testClient.Initialize(ctx, authorizationHeaders())
			resp := testClient.ListTools(ctx, authorizationHeaders())
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(verifier.calls).To(Equal(2))
		}, NodeTimeout(5*time.Second))

		DescribeTable("rejects invalid authorization headers",
			func(headers http.Header) {
				resp := testClient.RPC(context.Background(), "tools/list", nil, headers)
				defer func() { _ = resp.Body.Close() }()
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
				Expect(resp.Header.Get("WWW-Authenticate")).To(ContainSubstring("resource_metadata="))
			},
			Entry("missing", http.Header{}),
			Entry("wrong scheme", http.Header{"Authorization": []string{"Basic abc"}}),
			Entry("empty bearer", http.Header{"Authorization": []string{"Bearer"}}),
			Entry("duplicate", http.Header{"Authorization": []string{"Bearer one", "Bearer two"}}),
		)

		It("returns invalid_token when JWT verification fails", func() {
			headers := http.Header{"Authorization": []string{"Bearer wrong-token"}}
			resp := testClient.RPC(context.Background(), "tools/list", nil, headers)
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(resp.Header.Get("WWW-Authenticate")).To(ContainSubstring(`error="invalid_token"`))
		})

		It("returns insufficient_scope with 403 for a valid but under-scoped token", func() {
			headers := http.Header{"Authorization": []string{"Bearer insufficient-scope"}}
			resp := testClient.RPC(context.Background(), "tools/list", nil, headers)
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			Expect(resp.Header.Get("WWW-Authenticate")).To(ContainSubstring(`error="insufficient_scope"`))
			Expect(resp.Header.Get("WWW-Authenticate")).To(ContainSubstring(`scope="mcp:tools"`))
		})

		It("rejects non-allowlisted browser origins before token verification", func() {
			headers := authorizationHeaders()
			headers.Set("Origin", "https://evil.example.com")
			resp := testClient.RPC(context.Background(), "tools/list", nil, headers)
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			Expect(verifier.calls).To(BeZero())
		})

		It("returns CORS headers only for an exact allowlisted origin", func(ctx SpecContext) {
			headers := authorizationHeaders()
			headers.Set("Origin", allowedOrigin)
			testClient.Initialize(ctx, headers)
			resp := testClient.RPC(ctx, "ping", nil, headers)
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Access-Control-Allow-Origin")).To(Equal(allowedOrigin))
			Expect(resp.Header.Get("Vary")).To(ContainSubstring("Origin"))
		}, NodeTimeout(5*time.Second))

		It("answers an allowlisted CORS preflight without a token", func() {
			req := httptest.NewRequest(http.MethodOptions, "/", nil)
			req.Header.Set("Origin", allowedOrigin)
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			recorder := httptest.NewRecorder()
			testClient.handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNoContent))
			Expect(recorder.Header().Get("Access-Control-Allow-Origin")).To(Equal(allowedOrigin))
			Expect(recorder.Header().Get("Access-Control-Allow-Headers")).To(ContainSubstring("Authorization"))
			Expect(verifier.calls).To(BeZero())
		})

		It("publishes OAuth protected-resource metadata without authentication", func() {
			req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/sysdig-mcp-server", nil)
			recorder := httptest.NewRecorder()
			testClient.handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
			Expect(recorder.Body.String()).To(ContainSubstring(resourceURL))
			Expect(recorder.Body.String()).To(ContainSubstring("https://identity.example.com"))
			Expect(verifier.calls).To(BeZero())
		})

		It("never forwards the MCP access token to Sysdig", func(ctx SpecContext) {
			var upstreamAuthorization string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upstreamAuthorization = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"permissions":[]}`))
			}))
			defer upstream.Close()

			sysdigClient, err := sysdig.NewSysdigClient(sysdig.WithFixedHostAndToken(upstream.URL, "server-side-sysdig-token"))
			Expect(err).NotTo(HaveOccurred())
			isolatedHandler := localmcp.NewHandler("dev", sysdigClient)
			isolatedHandler.RegisterTools(&dummyTool{name: "tool1"})
			client := NewHTTPTestClient(isolatedHandler.AsStreamableHTTP("/", false, remoteSecurity(verifier)))

			client.Initialize(ctx, authorizationHeaders())
			resp := client.ListTools(ctx, authorizationHeaders())
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(upstreamAuthorization).To(Equal("Bearer server-side-sysdig-token"))
			Expect(upstreamAuthorization).NotTo(ContainSubstring(validMCPToken))
		}, NodeTimeout(5*time.Second))

		It("constructs a protected SSE handler", func() {
			Expect(handler.AsSSE("/sse", remoteSecurity(verifier))).NotTo(BeNil())
		})

		It("serves stateless calls without initialization", func(ctx SpecContext) {
			statelessClient := NewHTTPTestClient(handler.AsStreamableHTTP("/", true, remoteSecurity(verifier)))
			handler.RegisterTools(&dummyTool{name: "tool1"})
			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &sysdig.UserPermissions{Permissions: []string{}},
			}, nil)

			resp := statelessClient.ListTools(ctx, authorizationHeaders())
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Mcp-Session-Id")).To(BeEmpty())
		}, NodeTimeout(5*time.Second))
	})

	Context("Stdio", func() {
		It("returns when the context is cancelled", func(ctx SpecContext) {
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			reader, writer := io.Pipe()
			defer func() { _ = writer.Close() }()

			_ = handler.ServeStdio(cancelled, reader, io.Discard)
		}, NodeTimeout(5*time.Second))
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
	handler   http.Handler
	sessionID string
}

func NewHTTPTestClient(handler http.Handler) *HTTPTestClient {
	return &HTTPTestClient{handler: handler}
}

func (c *HTTPTestClient) RPC(ctx context.Context, method string, params any, headers http.Header) *http.Response {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, req)
	return recorder.Result()
}

func (c *HTTPTestClient) Initialize(ctx context.Context, headers http.Header) {
	params := mcp.InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: mcp.Implementation{
			Name:    "test",
			Version: "1.0",
		},
		Capabilities: mcp.ClientCapabilities{},
	}

	resp := c.RPC(ctx, "initialize", params, headers)
	defer func() { _ = resp.Body.Close() }()
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	c.sessionID = resp.Header.Get("Mcp-Session-Id")
}

func (c *HTTPTestClient) ListTools(ctx context.Context, headers http.Header) *http.Response {
	return c.RPC(ctx, "tools/list", nil, headers)
}
