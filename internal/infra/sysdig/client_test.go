package sysdig_test

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

var _ = Describe("Client TLS", func() {
	var ts *httptest.Server

	BeforeEach(func() {
		// Start a TLS server with a self-signed certificate
		ts = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/users/me/permissions" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"permissions":[]}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		// Redirect server logs to GinkgoWriter to avoid noise in test output
		ts.Config.ErrorLog = log.New(GinkgoWriter, "", 0)
	})

	AfterEach(func() {
		ts.Close()
	})

	It("should fail request with self-signed cert by default", func() {
		// Create client pointing to the TLS server without custom transport
		client, err := sysdig.NewSysdigClient(
			sysdig.WithFixedHostAndToken(ts.URL, "dummy-token"),
		)
		Expect(err).NotTo(HaveOccurred())

		// Attempt request
		_, err = client.GetMyPermissionsWithResponse(context.Background())
		Expect(err).To(HaveOccurred())
		// Verification that it failed due to certificate issues
		Expect(err.Error()).To(Or(ContainSubstring("certificate"), ContainSubstring("unknown authority")))
	})

	It("should succeed request when using custom HTTP client with InsecureSkipVerify", func() {
		// Create custom HTTP client that skips verification
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		httpClient := &http.Client{Transport: transport}

		// Create client using the custom HTTP client
		client, err := sysdig.NewSysdigClient(
			sysdig.WithFixedHostAndToken(ts.URL, "dummy-token"),
			sysdig.WithHTTPClient(httpClient),
		)
		Expect(err).NotTo(HaveOccurred())

		// Attempt request
		resp, err := client.GetMyPermissionsWithResponse(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
	})
})
