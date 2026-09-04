package mcp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

const corsMaxAgeSeconds = 600

type Handler struct {
	server *server.MCPServer
}

type mcpTool interface {
	RegisterInServer(server *server.MCPServer)
}

func toolPermissionFiltering(sysdigClient sysdig.ExtendedClientWithResponsesInterface) server.ToolFilterFunc {
	return func(ctx context.Context, mcpTools []mcp.Tool) []mcp.Tool {
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

		for _, tool := range mcpTools {
			requiredPermissions := tools.RequiredPermissionsFromTool(tool)
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

func NewHandler(version string, sysdigClient sysdig.ExtendedClientWithResponsesInterface) *Handler {
	s := server.NewMCPServer(
		"Sysdig MCP Server",
		version,
		server.WithInstructions("Provides read-only Sysdig Monitor tools for infrastructure analysis."),
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

func (h *Handler) AsStreamableHTTP(mountPath string, stateless bool, security RemoteSecurity) http.Handler {
	mux := http.NewServeMux()

	opts := []server.StreamableHTTPOption{
		server.WithStreamableHTTPCORS(remoteCORSOptions(security)...),
	}
	if stateless {
		opts = append(opts, server.WithStateLess(true))
	}

	httpServer := server.NewStreamableHTTPServer(h.server, opts...)
	security.mountMetadata(mux)
	mux.Handle(mountPath, security.protect(httpServer))
	return mux
}

func (h *Handler) AsSSE(mountPath string, security RemoteSecurity) http.Handler {
	mux := http.NewServeMux()
	sseServer := server.NewSSEServer(
		h.server,
		server.WithStaticBasePath(mountPath),
		server.WithSSECORS(remoteCORSOptions(security)...),
	)
	security.mountMetadata(mux)
	mux.Handle(sseServer.CompleteSsePath(), security.protect(sseServer.SSEHandler()))
	mux.Handle(sseServer.CompleteMessagePath(), security.protect(sseServer.MessageHandler()))
	return mux
}

func remoteCORSOptions(security RemoteSecurity) []server.CORSOption {
	return []server.CORSOption{
		server.WithCORSAllowedOrigins(security.corsOrigins()...),
		server.WithCORSAllowedMethods(http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions),
		server.WithCORSAllowedHeaders(
			"Authorization",
			"Content-Type",
			"Last-Event-ID",
			server.HeaderKeyProtocolVersion,
			server.HeaderKeySessionID,
		),
		server.WithCORSExposedHeaders(server.HeaderKeySessionID, "WWW-Authenticate"),
		server.WithCORSMaxAge(corsMaxAgeSeconds),
	}
}

func (h *Handler) ServeInProcessClient() (*client.Client, error) {
	return client.NewInProcessClient(h.server)
}
