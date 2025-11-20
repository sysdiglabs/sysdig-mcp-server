package mcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

type Handler struct {
	server *server.MCPServer
}

type mcpTool interface {
	RegisterInServer(server *server.MCPServer)
}

func toolPermissionFiltering(sysdigClient sysdig.ExtendedClientWithResponsesInterface) server.ToolFilterFunc {
	return func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		allowedTools := []mcp.Tool{}
		slog.Debug("filtering tools")

		response, err := sysdigClient.GetMyPermissionsWithResponse(ctx)
		if err != nil {
			slog.Error("unable to retrieve permissions to enable tools", "err", err.Error())
			return allowedTools
		}
		if response.JSON200 == nil {
			slog.Error("unable to retrieve permissions", "statusCode", response.HTTPResponse.StatusCode, "status", response.HTTPResponse.Status)
			return allowedTools
		}

		userPermissions := response.JSON200.Permissions
		userHasAllRequiredPermissions := func(requiredPermissions ...string) bool {
			for _, permission := range requiredPermissions {
				if !slices.Contains(userPermissions, permission) {
					return false
				}
			}
			return true
		}

		for _, tool := range tools {
			requiredPermissions := RequiredPermissionsFromTool(tool)
			if userHasAllRequiredPermissions(requiredPermissions...) {
				slog.Debug("tool meets all the required permissions", "name", tool.Name)
				allowedTools = append(allowedTools, tool)
				continue
			}

			slog.Debug("tool does not meet the required permissions, skipping it", "name", tool.Name, "requiredPermissions", strings.Join(requiredPermissions, ","), "userPermissions", strings.Join(userPermissions, ","))
		}

		return allowedTools
	}
}

func NewHandler(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *Handler {
	s := server.NewMCPServer(
		"Sysdig MCP Server",
		"1.0.0",
		server.WithInstructions("Provides Sysdig Secure tools and resources."),
		server.WithToolCapabilities(true),
		server.WithToolFilter(toolPermissionFiltering(sysdigClient)),
	)

	return &Handler{
		server: s,
	}
}

func (h *Handler) RegisterTools(tools ...mcpTool) {
	for _, tool := range tools {
		tool.RegisterInServer(h.server)
	}
}

func (h *Handler) ServeStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return server.NewStdioServer(h.server).Listen(ctx, stdin, stdout)
}

func (h *Handler) ServeStreamableHTTP(addr, mountPath string) error {
	fmt.Printf("MCP Server listening on %s%s\n", addr, mountPath)
	return http.ListenAndServe(addr, h.AsStreamableHTTP(mountPath))
}

func (h *Handler) AsStreamableHTTP(mountPath string) http.Handler {
	mux := http.NewServeMux()
	httpServer := server.NewStreamableHTTPServer(h.server)
	mux.Handle(mountPath, authMiddleware(httpServer))
	return mux
}

func (h *Handler) ServeSSE(addr, mountPath string) error {
	fmt.Printf("MCP Server listening on %s%s\n", addr, mountPath)
	return http.ListenAndServe(addr, h.AsSSE(mountPath))
}

func (h *Handler) AsSSE(mountPath string) http.Handler {
	mux := http.NewServeMux()
	sseServer := server.NewSSEServer(h.server, server.WithStaticBasePath(mountPath))
	mux.Handle(mountPath, authMiddleware(sseServer))
	return mux
}

func (h *Handler) ServeInProcessClient() (*client.Client, error) {
	return client.NewInProcessClient(h.server)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if host := r.Header.Get("X-Sysdig-Host"); host != "" {
			ctx = sysdig.WrapContextWithHost(ctx, host)
		}

		var token string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}

		if token == "" {
			token = r.Header.Get("X-Sysdig-Token")
		}

		if token != "" {
			ctx = sysdig.WrapContextWithToken(ctx, token)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
