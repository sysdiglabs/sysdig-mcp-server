package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/config"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		Context("with a valid config", func() {
			It("should not return an error", func() {
				cfg := &config.Config{
					APIHost:  "host",
					APIToken: "token",
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Context("with a missing api host", func() {
			It("should return an error if transport is stdio", func() {
				cfg := &config.Config{
					Transport: "stdio",
					APIToken:  "token",
				}
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("SYSDIG_MCP_API_HOST"))
			})

			It("should not return an error if transport is not stdio", func() {
				cfg := &config.Config{
					Transport: "sse",
					APIToken:  "token",
				}
				err := cfg.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with a missing api token", func() {
			It("should not return an error if transport is not stdio", func() {
				cfg := &config.Config{
					APIHost: "host",
				}
				err := cfg.Validate()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error if transport is stdio", func() {
				cfg := &config.Config{
					Transport: "stdio",
					APIHost:   "host",
				}
				err := cfg.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("SYSDIG_MCP_API_SECURE_TOKEN"))
			})
		})
	})

	Describe("Load", func() {
		BeforeEach(func() {
			os.Clearenv()
		})

		Context("with required env vars set for stdio", func() {
			BeforeEach(func() {
				_ = os.Setenv("SYSDIG_MCP_API_HOST", "host")
				_ = os.Setenv("SYSDIG_MCP_API_SECURE_TOKEN", "token")
			})

			It("should load default values", func() {
				cfg, err := config.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Transport).To(Equal("stdio"))
				Expect(cfg.ListeningHost).To(Equal("localhost"))
				Expect(cfg.ListeningPort).To(Equal("8080"))
				Expect(cfg.MountPath).To(Equal("/sysdig-mcp-server"))
				Expect(cfg.LogLevel).To(Equal("INFO"))
			})
		})

		Context("with required env vars set for http", func() {
			BeforeEach(func() {
				_ = os.Setenv("SYSDIG_MCP_API_HOST", "host")
				_ = os.Setenv("SYSDIG_MCP_TRANSPORT", "streamable-http")
			})

			It("should load default values", func() {
				cfg, err := config.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Transport).To(Equal("streamable-http"))
				Expect(cfg.ListeningHost).To(Equal("localhost"))
				Expect(cfg.ListeningPort).To(Equal("8080"))
				Expect(cfg.MountPath).To(Equal("/sysdig-mcp-server"))
				Expect(cfg.LogLevel).To(Equal("INFO"))
			})
		})

		Context("with all env vars set", func() {
			BeforeEach(func() {
				_ = os.Setenv("SYSDIG_MCP_API_HOST", "env-host")
				_ = os.Setenv("SYSDIG_MCP_API_SECURE_TOKEN", "env-token")
				_ = os.Setenv("SYSDIG_MCP_TRANSPORT", "http")
				_ = os.Setenv("SYSDIG_MCP_LISTENING_HOST", "0.0.0.0")
				_ = os.Setenv("SYSDIG_MCP_LISTENING_PORT", "9090")
				_ = os.Setenv("SYSDIG_MCP_MOUNT_PATH", "/custom")
				_ = os.Setenv("SYSDIG_MCP_LOGLEVEL", "DEBUG")
			})

			It("should load all values from the environment", func() {
				cfg, err := config.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.APIHost).To(Equal("env-host"))
				Expect(cfg.APIToken).To(Equal("env-token"))
				Expect(cfg.Transport).To(Equal("http"))
				Expect(cfg.ListeningHost).To(Equal("0.0.0.0"))
				Expect(cfg.ListeningPort).To(Equal("9090"))
				Expect(cfg.MountPath).To(Equal("/custom"))
				Expect(cfg.LogLevel).To(Equal("DEBUG"))
			})
		})

		Context("without required env vars", func() {
			It("should return an error", func() {
				_, err := config.Load()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
