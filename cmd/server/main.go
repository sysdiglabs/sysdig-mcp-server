package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/config"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

var Version = "dev"

func init() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:     "sysdig-mcp-server",
		Short:   "Sysdig MCP Server",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
		SilenceUsage: true,
	}

	rootCmd.SetVersionTemplate("{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
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
	sysdigClientOptions := []sysdig.IntoClientOption{
		sysdig.WithVersion(Version),
		sysdig.WithFallbackAuthentication(
			sysdig.WithHostAndTokenFromContext(),
			sysdig.WithFixedHostAndToken(cfg.APIHost, cfg.APIToken),
		),
	}

	if cfg.SkipTLSVerification {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
		httpClient := &http.Client{Transport: transport}

		sysdigClientOptions = append(sysdigClientOptions, sysdig.WithHTTPClient(httpClient))
	}

	sysdigClient, err := sysdig.NewSysdigClient(sysdigClientOptions...)
	if err != nil {
		return nil, fmt.Errorf("error creating sysdig client: %w", err)
	}
	return sysdigClient, nil
}

func setupHandler(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *mcp.Handler {
	systemClock := clock.NewSystemClock()
	handler := mcp.NewHandler(Version, sysdigClient)
	handler.RegisterTools(
		tools.NewToolListRuntimeEvents(sysdigClient, systemClock),
		tools.NewToolGetEventInfo(sysdigClient),
		tools.NewToolGetEventProcessTree(sysdigClient),
		tools.NewToolRunSysql(sysdigClient),
		tools.NewToolGenerateSysql(sysdigClient),

		tools.NewK8sListClusters(sysdigClient),
		tools.NewK8sListNodes(sysdigClient),
		tools.NewK8sListCronjobs(sysdigClient),
		tools.NewK8sListWorkloads(sysdigClient),
		tools.NewK8sListPodContainers(sysdigClient),
		tools.NewK8sListTopUnavailablePods(sysdigClient),
		tools.NewK8sListTopRestartedPods(sysdigClient),
		tools.NewK8sListTopHttpErrorsInPods(sysdigClient),
		tools.NewK8sListTopNetworkErrorsInPods(sysdigClient),
		tools.NewK8sListCountPodsPerCluster(sysdigClient),
		tools.NewK8sListUnderutilizedPodsCPUQuota(sysdigClient),
		tools.NewK8sListTopCPUConsumedWorkload(sysdigClient),
		tools.NewK8sListTopCPUConsumedContainer(sysdigClient),
		tools.NewK8sListUnderutilizedPodsMemoryQuota(sysdigClient),
		tools.NewK8sListTopMemoryConsumedWorkload(sysdigClient),
		tools.NewK8sListTopMemoryConsumedContainer(sysdigClient),
	)
	return handler
}

func startServer(cfg *config.Config, handler *mcp.Handler) error {
	switch cfg.Transport {
	case "stdio":
		if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			slog.Error("server error", "err", err)
		}
	case "streamable-http":
		addr := fmt.Sprintf("%s:%s", cfg.ListeningHost, cfg.ListeningPort)
		slog.Info("MCP Server listening", "addr", addr, "mountPath", cfg.MountPath)
		if err := http.ListenAndServe(addr, handler.AsStreamableHTTP(cfg.MountPath)); err != nil {
			return fmt.Errorf("error serving streamable http: %w", err)
		}
	case "sse":
		addr := fmt.Sprintf("%s:%s", cfg.ListeningHost, cfg.ListeningPort)
		slog.Info("MCP Server listening", "addr", addr, "mountPath", cfg.MountPath)
		if err := http.ListenAndServe(addr, handler.AsSSE(cfg.MountPath)); err != nil {
			return fmt.Errorf("error serving sse: %w", err)
		}
	default:
		return fmt.Errorf("unknown transport: %s", cfg.Transport)
	}
	return nil
}
