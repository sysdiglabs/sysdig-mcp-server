package mcp

import (
	"context"
	"io"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/server"
)

type Handler struct {
	server *server.MCPServer
}

type mcpTool interface {
	RegisterInServer(server *server.MCPServer)
	CanBeUsed() bool
}

func NewHandler() *Handler {
	s := server.NewMCPServer(
		"Sysdig MCP Server",
		"1.0.0",
		server.WithInstructions("Provides Sysdig Secure tools and resources."),
		server.WithToolCapabilities(true),
	)
	return &Handler{
		server: s,
	}
}

func NewHandlerWithTools(tools ...mcpTool) *Handler {
	h := NewHandler()
	h.RegisterTools(tools...)
	return h
}

func (h *Handler) RegisterTools(tools ...mcpTool) {
	for _, tool := range tools {
		if tool.CanBeUsed() {
			tool.RegisterInServer(h.server)
		}
	}
}

func (h *Handler) ServeStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return server.NewStdioServer(h.server).Listen(ctx, stdin, stdout)
}

func (h *Handler) ServeInProcessClient() (*client.Client, error) {
	return client.NewInProcessClient(h.server)
}
