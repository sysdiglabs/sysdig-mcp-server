package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/config"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	setupLogger(cfg.LogLevel)

	sysdigClient, err := setupSysdigClient(cfg)
	if err != nil {
		return err
	}

	handler := setupHandler(sysdigClient)

	return startServer(cfg, handler)
}

func setupLogger(logLevel string) {
	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}

func setupSysdigClient(cfg *config.Config) (sysdig.ExtendedClientWithResponsesInterface, error) {
	sysdigClient, err := sysdig.NewSysdigClient(cfg.APIHost, cfg.APIToken)
	if err != nil {
		return nil, fmt.Errorf("error creating sysdig client: %w", err)
	}
	return sysdigClient, nil
}

func setupHandler(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *mcp.Handler {
	systemClock := clock.NewSystemClock()
	permissionChecker := mcp.NewPermissionChecker(sysdigClient)

	return mcp.NewHandlerWithTools(
		mcp.NewToolListRuntimeEvents(sysdigClient, systemClock, permissionChecker),
		mcp.NewToolGetEventInfo(sysdigClient, permissionChecker),
		mcp.NewToolGetEventProcessTree(sysdigClient, permissionChecker),
		mcp.NewToolRunSysql(sysdigClient, permissionChecker),
		mcp.NewToolGenerateSysql(sysdigClient, permissionChecker),
	)
}

func startServer(cfg *config.Config, handler *mcp.Handler) error {
	switch cfg.Transport {
	case "stdio":
		if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			// Stdio server errors are not fatal for the process, just print them
			fmt.Printf("Server error: %v\n", err)
		}
	case "streamable-http":
		addr := fmt.Sprintf("%s:%s", cfg.ListeningHost, cfg.ListeningPort)
		if err := handler.ServeStreamableHTTP(addr, cfg.MountPath); err != nil {
			return fmt.Errorf("error serving streamable http: %w", err)
		}
	case "sse":
		addr := fmt.Sprintf("%s:%s", cfg.ListeningHost, cfg.ListeningPort)
		if err := handler.ServeSSE(addr, cfg.MountPath); err != nil {
			return fmt.Errorf("error serving sse: %w", err)
		}
	default:
		return fmt.Errorf("unknown transport: %s", cfg.Transport)
	}
	return nil
}
