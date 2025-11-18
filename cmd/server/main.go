package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

func main() {
	apiHost := os.Getenv("SYSDIG_MCP_API_HOST")
	if apiHost == "" {
		slog.Error("SYSDIG_MCP_API_HOST env var is empty or not set")
		os.Exit(1)
	}
	apiToken := os.Getenv("SYSDIG_MCP_API_SECURE_TOKEN")
	if apiToken == "" {
		slog.Error("SYSDIG_MCP_API_SECURE_TOKEN env var is empty or not set")
		os.Exit(1)
	}

	sysdigClient, err := sysdig.NewSysdigClient(apiHost, apiToken)
	if err != nil {
		slog.Error("error creating sysdig client", "error", err.Error())
		os.Exit(1)
	}
	systemClock := clock.NewSystemClock()
	permissionChecker := mcp.NewPermissionChecker(sysdigClient)

	handler := mcp.NewHandlerWithTools(
		mcp.NewToolListRuntimeEvents(sysdigClient, systemClock, permissionChecker),
		mcp.NewToolGetEventInfo(sysdigClient, permissionChecker),
		mcp.NewToolGetEventProcessTree(sysdigClient, permissionChecker),
		mcp.NewToolRunSysql(sysdigClient, permissionChecker),
		mcp.NewToolGenerateSysql(sysdigClient, permissionChecker),
	)

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	transport := os.Getenv("SYSDIG_MCP_TRANSPORT")
	if transport == "" {
		transport = "stdio"
	}

	switch transport {
	case "stdio":
		if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	case "streamable-http":
		host := os.Getenv("SYSDIG_MCP_LISTENING_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("SYSDIG_MCP_LISTENING_PORT")
		if port == "" {
			port = "8080"
		}
		mountPath := os.Getenv("SYSDIG_MCP_MOUNT_PATH")
		if mountPath == "" {
			mountPath = "/sysdig-mcp-server"
		}
		addr := fmt.Sprintf("%s:%s", host, port)
		if err := handler.ServeStreamableHTTP(addr, mountPath); err != nil {
			slog.Error("error serving streamable http", "error", err.Error())
			os.Exit(1)
		}
	case "sse":
		host := os.Getenv("SYSDIG_MCP_LISTENING_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("SYSDIG_MCP_LISTENING_PORT")
		if port == "" {
			port = "8080"
		}
		mountPath := os.Getenv("SYSDIG_MCP_MOUNT_PATH")
		if mountPath == "" {
			mountPath = "/sysdig-mcp-server"
		}
		addr := fmt.Sprintf("%s:%s", host, port)
		if err := handler.ServeSSE(addr, mountPath); err != nil {
			slog.Error("error serving sse", "error", err.Error())
			os.Exit(1)
		}
	}
}
