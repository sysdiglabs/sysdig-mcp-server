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
	if apiHost == "" {
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
	)

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
