package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	APIHost             string
	APIToken            string
	SkipTLSVerification bool
	Transport           string
	ListeningHost       string
	ListeningPort       string
	MountPath           string
	LogLevel            string
}

func (c *Config) Validate() error {
	if c.Transport == "stdio" && c.APIHost == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_API_HOST")
	}
	if c.Transport == "stdio" && c.APIToken == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_API_TOKEN")
	}
	return nil
}

func Load() (*Config, error) {
	cfg := &Config{
		APIHost:             getEnv("SYSDIG_MCP_API_HOST", ""),
		APIToken:            getEnv("SYSDIG_MCP_API_TOKEN", ""),
		SkipTLSVerification: getEnv("SYSDIG_MCP_API_SKIP_TLS_VERIFICATION", false),
		Transport:           getEnv("SYSDIG_MCP_TRANSPORT", "stdio"),
		ListeningHost:       getEnv("SYSDIG_MCP_LISTENING_HOST", "localhost"),
		ListeningPort:       getEnv("SYSDIG_MCP_LISTENING_PORT", "8080"),
		MountPath:           getEnv("SYSDIG_MCP_MOUNT_PATH", "/sysdig-mcp-server"),
		LogLevel:            getEnv("SYSDIG_MCP_LOGLEVEL", "INFO"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

type envType interface {
	~string | ~bool
}

func getEnv[T envType](key string, fallback T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	switch any(fallback).(type) {
	case string:
		return any(value).(T)

	case bool:
		value = strings.TrimSpace(value)
		if value == "" {
			return fallback
		}

		b, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}
		return any(b).(T)
	}

	return fallback
}
