package config

import (
	"fmt"
	"os"
)

type Config struct {
	APIHost       string
	APIToken      string
	Transport     string
	ListeningHost string
	ListeningPort string
	MountPath     string
	LogLevel      string
}

func (c *Config) Validate() error {
	if c.APIHost == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_API_HOST")
	}
	if c.APIToken == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_API_SECURE_TOKEN")
	}
	return nil
}

func Load() (*Config, error) {
	cfg := &Config{
		APIHost:       getEnv("SYSDIG_MCP_API_HOST", ""),
		APIToken:      getEnv("SYSDIG_MCP_API_SECURE_TOKEN", ""),
		Transport:     getEnv("SYSDIG_MCP_TRANSPORT", "stdio"),
		ListeningHost: getEnv("SYSDIG_MCP_LISTENING_HOST", "localhost"),
		ListeningPort: getEnv("SYSDIG_MCP_LISTENING_PORT", "8080"),
		MountPath:     getEnv("SYSDIG_MCP_MOUNT_PATH", "/sysdig-mcp-server"),
		LogLevel:      getEnv("SYSDIG_MCP_LOGLEVEL", "INFO"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
