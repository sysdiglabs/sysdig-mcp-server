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
	"time"

	"github.com/spf13/cobra"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/config"
	infraauth "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/auth"
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
		sysdig.WithFixedHostAndToken(cfg.APIHost, cfg.APIToken),
	}

	if cfg.SkipTLSVerification {
		sysdigClientOptions = append(sysdigClientOptions, sysdig.WithHTTPClient(insecureHTTPClient()))
	}

	sysdigClient, err := sysdig.NewSysdigClient(sysdigClientOptions...)
	if err != nil {
		return nil, fmt.Errorf("error creating sysdig client: %w", err)
	}
	return sysdigClient, nil
}

func setupRemoteSecurity(cfg *config.Config) mcp.RemoteSecurity {
	var jwksHTTPClient *http.Client
	if cfg.SkipJWKSTLSVerification {
		jwksHTTPClient = insecureHTTPClient()
	}

	verifier := infraauth.NewJWTVerifierWithHTTPClient(
		context.Background(),
		cfg.AuthIssuer,
		cfg.ResourceURL,
		cfg.AuthJWKSURL,
		cfg.AuthSigningAlgs,
		cfg.AuthScopes,
		jwksHTTPClient,
	)

	return mcp.NewRemoteSecurity(
		verifier,
		cfg.ResourceURL,
		cfg.AuthIssuer,
		cfg.AuthScopes,
		cfg.AllowedOrigins,
	)
}

func insecureHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = true
	return &http.Client{Transport: transport}
}

func setupHandler(sysdigClient sysdig.ExtendedClientWithResponsesInterface) *mcp.Handler {
	systemClock := clock.NewSystemClock()
	handler := mcp.NewHandler(Version, sysdigClient)
	handler.RegisterTools(
		tools.NewK8sListClusters(sysdigClient, systemClock),
		tools.NewK8sListNodes(sysdigClient, systemClock),
		tools.NewK8sListCronjobs(sysdigClient, systemClock),
		tools.NewK8sListWorkloads(sysdigClient, systemClock),
		tools.NewK8sListPodContainers(sysdigClient, systemClock),
		tools.NewK8sListTopUnavailablePods(sysdigClient, systemClock),
		tools.NewK8sListTopRestartedPods(sysdigClient, systemClock),
		tools.NewK8sListTopHttpErrorsInPods(sysdigClient, systemClock),
		tools.NewK8sListTopNetworkErrorsInPods(sysdigClient, systemClock),
		tools.NewK8sListCountPodsPerCluster(sysdigClient, systemClock),
		tools.NewK8sListUnderutilizedPodsCPUQuota(sysdigClient, systemClock),
		tools.NewK8sListTopCPUConsumedWorkload(sysdigClient, systemClock),
		tools.NewK8sListTopCPUConsumedContainer(sysdigClient, systemClock),
		tools.NewK8sListUnderutilizedPodsMemoryQuota(sysdigClient, systemClock),
		tools.NewK8sListTopMemoryConsumedWorkload(sysdigClient, systemClock),
		tools.NewK8sListTopMemoryConsumedContainer(sysdigClient, systemClock),
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
		slog.Info("MCP Server listening", "addr", addr, "mountPath", cfg.MountPath, "stateless", cfg.Stateless)
		server := &http.Server{
			Addr:              addr,
			Handler:           handler.AsStreamableHTTP(cfg.MountPath, cfg.Stateless, setupRemoteSecurity(cfg)),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			IdleTimeout:       2 * time.Minute,
		}
		if err := server.ListenAndServe(); err != nil {
			return fmt.Errorf("error serving streamable http: %w", err)
		}
	case "sse":
		addr := fmt.Sprintf("%s:%s", cfg.ListeningHost, cfg.ListeningPort)
		slog.Info("MCP Server listening", "addr", addr, "mountPath", cfg.MountPath)
		server := &http.Server{
			Addr:              addr,
			Handler:           handler.AsSSE(cfg.MountPath, setupRemoteSecurity(cfg)),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			IdleTimeout:       2 * time.Minute,
		}
		if err := server.ListenAndServe(); err != nil {
			return fmt.Errorf("error serving sse: %w", err)
		}
	default:
		return fmt.Errorf("unknown transport: %s", cfg.Transport)
	}
	return nil
}
