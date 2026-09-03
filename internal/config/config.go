package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
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
	Stateless           bool
	ResourceURL         string
	AuthIssuer          string
	AuthJWKSURL         string
	AuthScopes          []string
	AuthSigningAlgs     []string
	AllowedOrigins      []string
}

func (c *Config) Validate() error {
	if !slices.Contains([]string{"stdio", "streamable-http", "sse"}, c.Transport) {
		return fmt.Errorf("unsupported SYSDIG_MCP_TRANSPORT %q", c.Transport)
	}
	if c.APIHost == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_API_HOST")
	}
	if c.APIToken == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_API_TOKEN")
	}
	if err := validateAbsoluteURL("SYSDIG_MCP_API_HOST", c.APIHost); err != nil {
		return err
	}
	if c.Transport == "stdio" {
		return nil
	}

	if c.ResourceURL == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_RESOURCE_URL")
	}
	if c.AuthIssuer == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_AUTH_ISSUER")
	}
	if c.AuthJWKSURL == "" {
		return fmt.Errorf("required configuration missing: SYSDIG_MCP_AUTH_JWKS_URL")
	}
	if err := validateAbsoluteURL("SYSDIG_MCP_RESOURCE_URL", c.ResourceURL); err != nil {
		return err
	}
	if err := validateAbsoluteURL("SYSDIG_MCP_AUTH_ISSUER", c.AuthIssuer); err != nil {
		return err
	}
	if err := validateAbsoluteURL("SYSDIG_MCP_AUTH_JWKS_URL", c.AuthJWKSURL); err != nil {
		return err
	}
	for _, origin := range c.AllowedOrigins {
		if err := validateOrigin(origin); err != nil {
			return fmt.Errorf("invalid SYSDIG_MCP_ALLOWED_ORIGINS entry %q: %w", origin, err)
		}
	}
	if len(c.AuthSigningAlgs) == 0 {
		return fmt.Errorf("SYSDIG_MCP_AUTH_SIGNING_ALGS must contain at least one asymmetric signing algorithm")
	}
	for _, algorithm := range c.AuthSigningAlgs {
		if !slices.Contains([]string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512", "EdDSA"}, algorithm) {
			return fmt.Errorf("unsupported asymmetric signing algorithm %q in SYSDIG_MCP_AUTH_SIGNING_ALGS", algorithm)
		}
	}
	return nil
}

func Load() (*Config, error) {
	cfg := &Config{
		APIHost:             getEnv("SYSDIG_MCP_API_HOST", ""),
		APIToken:            getEnv("SYSDIG_MCP_API_TOKEN", ""),
		SkipTLSVerification: getEnv("SYSDIG_MCP_API_SKIP_TLS_VERIFICATION", false),
		Transport:           getEnv("SYSDIG_MCP_TRANSPORT", "stdio"),
		ListeningHost:       getEnv("SYSDIG_MCP_LISTENING_HOST", ""),
		ListeningPort:       getEnv("SYSDIG_MCP_LISTENING_PORT", "8080"),
		MountPath:           getEnv("SYSDIG_MCP_MOUNT_PATH", "/sysdig-mcp-server"),
		LogLevel:            getEnv("SYSDIG_MCP_LOGLEVEL", "INFO"),
		Stateless:           getEnv("SYSDIG_MCP_STATELESS", false),
		ResourceURL:         getEnv("SYSDIG_MCP_RESOURCE_URL", ""),
		AuthIssuer:          getEnv("SYSDIG_MCP_AUTH_ISSUER", ""),
		AuthJWKSURL:         getEnv("SYSDIG_MCP_AUTH_JWKS_URL", ""),
		AuthScopes:          getEnvList("SYSDIG_MCP_AUTH_SCOPES", nil),
		AuthSigningAlgs:     getEnvList("SYSDIG_MCP_AUTH_SIGNING_ALGS", []string{"RS256"}),
		AllowedOrigins:      getEnvList("SYSDIG_MCP_ALLOWED_ORIGINS", nil),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnvList(key string, fallback []string) []string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return slices.Clone(fallback)
	}

	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func validateAbsoluteURL(name, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	if u.User != nil || u.Fragment != "" {
		return fmt.Errorf("%s must not contain user information or a fragment", name)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHostname(u.Hostname())) {
		return fmt.Errorf("%s must use https (http is allowed only for loopback development)", name)
	}
	return nil
}

func validateOrigin(origin string) error {
	if origin == "*" {
		return fmt.Errorf("wildcard origins are not allowed")
	}
	u, err := url.Parse(origin)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("origin must be an absolute URL")
	}
	if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("origin must contain only scheme and authority")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHostname(u.Hostname())) {
		return fmt.Errorf("origin must use https (http is allowed only for loopback development)")
	}
	return nil
}

func isLoopbackHostname(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
