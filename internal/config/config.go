package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
)

var supportedAuthSigningAlgorithms = []string{
	oidc.RS256,
	oidc.RS384,
	oidc.RS512,
	oidc.PS256,
	oidc.PS384,
	oidc.PS512,
	oidc.ES256,
	oidc.ES384,
	oidc.ES512,
	oidc.EdDSA,
}

type Config struct {
	APIHost                 string
	APIToken                string
	SkipTLSVerification     bool
	SkipJWKSTLSVerification bool
	Transport               string
	ListeningHost           string
	ListeningPort           string
	MountPath               string
	LogLevel                string
	Stateless               bool
	ResourceURL             string
	AuthIssuer              string
	AuthJWKSURL             string
	AuthScopes              []string
	AuthSigningAlgs         []string
	AllowedOrigins          []string
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
	apiHost, err := parseAbsoluteURL("SYSDIG_MCP_API_HOST", c.APIHost)
	if err != nil {
		return err
	}
	if apiHost.RawQuery != "" {
		return fmt.Errorf("SYSDIG_MCP_API_HOST must not contain a query string")
	}
	if c.Transport == "stdio" {
		return nil
	}

	if err := validateMountPath(c.MountPath); err != nil {
		return err
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

	resourceURL, err := parseAbsoluteURL("SYSDIG_MCP_RESOURCE_URL", c.ResourceURL)
	if err != nil {
		return err
	}
	if resourceURL.RawQuery != "" {
		return fmt.Errorf("SYSDIG_MCP_RESOURCE_URL must not contain a query string")
	}
	resourcePath := resourceURL.Path
	if resourcePath == "" {
		resourcePath = "/"
	}
	if resourcePath != c.MountPath {
		return fmt.Errorf(
			"SYSDIG_MCP_RESOURCE_URL path %q must match SYSDIG_MCP_MOUNT_PATH %q",
			resourcePath,
			c.MountPath,
		)
	}

	authIssuer, err := parseAbsoluteURL("SYSDIG_MCP_AUTH_ISSUER", c.AuthIssuer)
	if err != nil {
		return err
	}
	if authIssuer.RawQuery != "" {
		return fmt.Errorf("SYSDIG_MCP_AUTH_ISSUER must not contain a query string")
	}
	if _, err := parseAbsoluteURL("SYSDIG_MCP_AUTH_JWKS_URL", c.AuthJWKSURL); err != nil {
		return err
	}
	for _, origin := range c.AllowedOrigins {
		if err := validateOrigin(origin); err != nil {
			return fmt.Errorf("invalid SYSDIG_MCP_ALLOWED_ORIGINS entry %q: %w", origin, err)
		}
	}
	for _, scope := range c.AuthScopes {
		if !validScopeToken(scope) {
			return fmt.Errorf("invalid scope %q in SYSDIG_MCP_AUTH_SCOPES", scope)
		}
	}
	if len(c.AuthSigningAlgs) == 0 {
		return fmt.Errorf("SYSDIG_MCP_AUTH_SIGNING_ALGS must contain at least one asymmetric signing algorithm")
	}
	for _, algorithm := range c.AuthSigningAlgs {
		if !slices.Contains(supportedAuthSigningAlgorithms, algorithm) {
			return fmt.Errorf("unsupported asymmetric signing algorithm %q in SYSDIG_MCP_AUTH_SIGNING_ALGS", algorithm)
		}
	}
	return nil
}

func Load() (*Config, error) {
	cfg := &Config{
		APIHost:                 getEnv("SYSDIG_MCP_API_HOST", ""),
		APIToken:                getEnv("SYSDIG_MCP_API_TOKEN", ""),
		SkipTLSVerification:     getEnv("SYSDIG_MCP_API_SKIP_TLS_VERIFICATION", false),
		SkipJWKSTLSVerification: getEnv("SYSDIG_MCP_AUTH_JWKS_SKIP_TLS_VERIFICATION", false),
		Transport:               getEnv("SYSDIG_MCP_TRANSPORT", "stdio"),
		ListeningHost:           getEnv("SYSDIG_MCP_LISTENING_HOST", ""),
		ListeningPort:           getEnv("SYSDIG_MCP_LISTENING_PORT", "8080"),
		MountPath:               getEnv("SYSDIG_MCP_MOUNT_PATH", "/sysdig-mcp-server"),
		LogLevel:                getEnv("SYSDIG_MCP_LOGLEVEL", "INFO"),
		Stateless:               getEnv("SYSDIG_MCP_STATELESS", false),
		ResourceURL:             getEnv("SYSDIG_MCP_RESOURCE_URL", ""),
		AuthIssuer:              getEnv("SYSDIG_MCP_AUTH_ISSUER", ""),
		AuthJWKSURL:             getEnv("SYSDIG_MCP_AUTH_JWKS_URL", ""),
		AuthScopes:              getEnvList("SYSDIG_MCP_AUTH_SCOPES", nil),
		AuthSigningAlgs:         getEnvList("SYSDIG_MCP_AUTH_SIGNING_ALGS", []string{oidc.RS256}),
		AllowedOrigins:          getEnvList("SYSDIG_MCP_ALLOWED_ORIGINS", nil),
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
		return r == ',' || unicode.IsSpace(r)
	})
}

func parseAbsoluteURL(name, rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return nil, fmt.Errorf("%s must be an absolute URL", name)
	}
	if u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain user information or a fragment", name)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHostname(u.Hostname())) {
		return nil, fmt.Errorf("%s must use https (http is allowed only for loopback development)", name)
	}
	return u, nil
}

func validateMountPath(mountPath string) error {
	if mountPath == "" || !strings.HasPrefix(mountPath, "/") || path.Clean(mountPath) != mountPath {
		return fmt.Errorf("SYSDIG_MCP_MOUNT_PATH must be an absolute, canonical URL path")
	}
	return nil
}

func validateOrigin(origin string) error {
	if origin == "*" {
		return fmt.Errorf("wildcard origins are not allowed")
	}
	u, err := parseAbsoluteURL("origin", origin)
	if err != nil {
		return err
	}
	if u.Path != "" || u.RawQuery != "" {
		return fmt.Errorf("origin must contain only scheme and authority")
	}
	return nil
}

func validScopeToken(scope string) bool {
	if scope == "" {
		return false
	}
	for _, r := range scope {
		// RFC 6749 scope-token = 1*( %x21 / %x23-5B / %x5D-7E ).
		if r < 0x21 || r > 0x7e || r == '"' || r == '\\' {
			return false
		}
	}
	return true
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
