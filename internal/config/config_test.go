package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/config"
)

func validConfig(transport string) *config.Config {
	cfg := &config.Config{
		APIHost:   "https://app.us4.sysdig.com",
		APIToken:  "sysdig-token",
		Transport: transport,
	}
	if transport != "stdio" {
		cfg.ResourceURL = "https://mcp.example.com/sysdig-mcp-server"
		cfg.AuthIssuer = "https://identity.example.com"
		cfg.AuthJWKSURL = "https://identity.example.com/.well-known/jwks.json"
		cfg.AuthSigningAlgs = []string{"RS256"}
		cfg.AllowedOrigins = []string{"https://client.example.com"}
	}
	return cfg
}

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		It("accepts stdio and secured remote configurations", func() {
			Expect(validConfig("stdio").Validate()).To(Succeed())
			Expect(validConfig("streamable-http").Validate()).To(Succeed())
			Expect(validConfig("sse").Validate()).To(Succeed())
		})

		DescribeTable("rejects missing credentials for every transport",
			func(transport string) {
				cfg := validConfig(transport)
				cfg.APIHost = ""
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("SYSDIG_MCP_API_HOST")))

				cfg = validConfig(transport)
				cfg.APIToken = ""
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("SYSDIG_MCP_API_TOKEN")))
			},
			Entry("stdio", "stdio"),
			Entry("streamable HTTP", "streamable-http"),
			Entry("SSE", "sse"),
		)

		It("rejects unsupported transports", func() {
			cfg := validConfig("websocket")
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("unsupported SYSDIG_MCP_TRANSPORT")))
		})

		DescribeTable("requires the remote OAuth configuration",
			func(clear func(*config.Config), expected string) {
				cfg := validConfig("streamable-http")
				clear(cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(expected)))
			},
			Entry("resource URL", func(cfg *config.Config) { cfg.ResourceURL = "" }, "SYSDIG_MCP_RESOURCE_URL"),
			Entry("issuer", func(cfg *config.Config) { cfg.AuthIssuer = "" }, "SYSDIG_MCP_AUTH_ISSUER"),
			Entry("JWKS URL", func(cfg *config.Config) { cfg.AuthJWKSURL = "" }, "SYSDIG_MCP_AUTH_JWKS_URL"),
		)

		DescribeTable("rejects unsafe URLs",
			func(mutate func(*config.Config), expected string) {
				cfg := validConfig("streamable-http")
				mutate(cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(expected)))
			},
			Entry("relative API host", func(cfg *config.Config) { cfg.APIHost = "app.example.com" }, "absolute URL"),
			Entry("plaintext resource", func(cfg *config.Config) { cfg.ResourceURL = "http://mcp.example.com" }, "must use https"),
			Entry("issuer with user info", func(cfg *config.Config) { cfg.AuthIssuer = "https://user@identity.example.com" }, "user information"),
			Entry("fragmented JWKS URL", func(cfg *config.Config) { cfg.AuthJWKSURL += "#keys" }, "fragment"),
			Entry("wildcard origin", func(cfg *config.Config) { cfg.AllowedOrigins = []string{"*"} }, "wildcard"),
			Entry("origin path", func(cfg *config.Config) { cfg.AllowedOrigins = []string{"https://client.example.com/path"} }, "scheme and authority"),
			Entry("empty signing algorithms", func(cfg *config.Config) { cfg.AuthSigningAlgs = nil }, "at least one"),
			Entry("symmetric signing", func(cfg *config.Config) { cfg.AuthSigningAlgs = []string{"HS256"} }, "asymmetric signing algorithm"),
		)

		It("allows HTTP only for loopback development", func() {
			cfg := validConfig("streamable-http")
			cfg.APIHost = "http://127.0.0.1:9000"
			cfg.ResourceURL = "http://localhost:8080/sysdig-mcp-server"
			cfg.AuthIssuer = "http://[::1]:9001"
			cfg.AuthJWKSURL = "http://localhost:9001/jwks"
			cfg.AllowedOrigins = []string{"http://localhost:5173"}
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Load", func() {
		BeforeEach(func() {
			os.Clearenv()
			_ = os.Setenv("SYSDIG_MCP_API_HOST", "https://app.us4.sysdig.com")
			_ = os.Setenv("SYSDIG_MCP_API_TOKEN", "sysdig-token")
		})

		It("loads stdio defaults", func() {
			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Transport).To(Equal("stdio"))
			Expect(cfg.ListeningHost).To(BeEmpty())
			Expect(cfg.ListeningPort).To(Equal("8080"))
			Expect(cfg.MountPath).To(Equal("/sysdig-mcp-server"))
			Expect(cfg.LogLevel).To(Equal("INFO"))
			Expect(cfg.SkipTLSVerification).To(BeFalse())
			Expect(cfg.Stateless).To(BeFalse())
			Expect(cfg.AuthSigningAlgs).To(Equal([]string{"RS256"}))
		})

		It("loads all remote security values", func() {
			_ = os.Setenv("SYSDIG_MCP_API_SKIP_TLS_VERIFICATION", "true")
			_ = os.Setenv("SYSDIG_MCP_TRANSPORT", "streamable-http")
			_ = os.Setenv("SYSDIG_MCP_LISTENING_HOST", "0.0.0.0")
			_ = os.Setenv("SYSDIG_MCP_LISTENING_PORT", "9090")
			_ = os.Setenv("SYSDIG_MCP_MOUNT_PATH", "/custom")
			_ = os.Setenv("SYSDIG_MCP_LOGLEVEL", "DEBUG")
			_ = os.Setenv("SYSDIG_MCP_STATELESS", "true")
			_ = os.Setenv("SYSDIG_MCP_RESOURCE_URL", "https://mcp.example.com/custom")
			_ = os.Setenv("SYSDIG_MCP_AUTH_ISSUER", "https://identity.example.com")
			_ = os.Setenv("SYSDIG_MCP_AUTH_JWKS_URL", "https://identity.example.com/jwks")
			_ = os.Setenv("SYSDIG_MCP_AUTH_SCOPES", "mcp:tools, profile")
			_ = os.Setenv("SYSDIG_MCP_AUTH_SIGNING_ALGS", "RS256 ES256")
			_ = os.Setenv("SYSDIG_MCP_ALLOWED_ORIGINS", "https://one.example.com, https://two.example.com")

			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.SkipTLSVerification).To(BeTrue())
			Expect(cfg.Transport).To(Equal("streamable-http"))
			Expect(cfg.ListeningHost).To(Equal("0.0.0.0"))
			Expect(cfg.ListeningPort).To(Equal("9090"))
			Expect(cfg.MountPath).To(Equal("/custom"))
			Expect(cfg.LogLevel).To(Equal("DEBUG"))
			Expect(cfg.Stateless).To(BeTrue())
			Expect(cfg.ResourceURL).To(Equal("https://mcp.example.com/custom"))
			Expect(cfg.AuthIssuer).To(Equal("https://identity.example.com"))
			Expect(cfg.AuthJWKSURL).To(Equal("https://identity.example.com/jwks"))
			Expect(cfg.AuthScopes).To(Equal([]string{"mcp:tools", "profile"}))
			Expect(cfg.AuthSigningAlgs).To(Equal([]string{"RS256", "ES256"}))
			Expect(cfg.AllowedOrigins).To(Equal([]string{"https://one.example.com", "https://two.example.com"}))
		})

		It("requires all remote settings", func() {
			_ = os.Setenv("SYSDIG_MCP_TRANSPORT", "sse")
			_, err := config.Load()
			Expect(err).To(MatchError(ContainSubstring("SYSDIG_MCP_RESOURCE_URL")))
		})

		It("falls back for invalid booleans", func() {
			_ = os.Setenv("SYSDIG_MCP_API_SKIP_TLS_VERIFICATION", "invalid-bool")
			cfg, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.SkipTLSVerification).To(BeFalse())
		})

		It("fails without required values", func() {
			os.Clearenv()
			_, err := config.Load()
			Expect(err).To(HaveOccurred())
		})
	})
})
