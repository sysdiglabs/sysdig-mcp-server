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
	sysdigClient, err := sysdig.NewSysdigClient(os.Getenv("SYSDIG_MCP_API_HOST"), os.Getenv("SYSDIG_MCP_API_SECURE_TOKEN"))
	if err != nil {
		slog.Error("error creating sysdig client", "error", err.Error())
		os.Exit(1)
	}
	systemClock := clock.NewSystemClock()

	handler := mcp.NewHandlerWithTools(
		mcp.NewToolListRuntimeEvents(sysdigClient, systemClock),
	)

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
