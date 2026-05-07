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

func permissionsHandler(captureHeaders func(http.Header)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captureHeaders != nil {
			captureHeaders(r.Header)
		}
		if r.URL.Path == "/api/users/me/permissions" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"permissions":[]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

var _ = Describe("Client TLS", func() {
	var ts *httptest.Server

	BeforeEach(func() {
		ts = httptest.NewTLSServer(permissionsHandler(nil))
		ts.Config.ErrorLog = log.New(GinkgoWriter, "", 0)
	})

	AfterEach(func() {
		ts.Close()
	})

	It("should fail request with self-signed cert by default", func() {
		client, err := sysdig.NewSysdigClient(
			sysdig.WithFixedHostAndToken(ts.URL, "dummy-token"),
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetMyPermissionsWithResponse(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Or(ContainSubstring("certificate"), ContainSubstring("unknown authority")))
	})

	It("should succeed request when using custom HTTP client with InsecureSkipVerify", func() {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		httpClient := &http.Client{Transport: transport}

		client, err := sysdig.NewSysdigClient(
			sysdig.WithFixedHostAndToken(ts.URL, "dummy-token"),
			sysdig.WithHTTPClient(httpClient),
		)
		Expect(err).NotTo(HaveOccurred())

		resp, err := client.GetMyPermissionsWithResponse(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
	})
})

var _ = Describe("Context helpers", func() {
	It("roundtrips token through context", func() {
		ctx := sysdig.WrapContextWithToken(context.Background(), "my-token")
		Expect(sysdig.GetTokenFromContext(ctx)).To(Equal("my-token"))
	})

	It("roundtrips host through context", func() {
		ctx := sysdig.WrapContextWithHost(context.Background(), "https://example.com")
		Expect(sysdig.GetHostFromContext(ctx)).To(Equal("https://example.com"))
	})

	It("panics when token is missing from context", func() {
		Expect(func() {
			sysdig.GetTokenFromContext(context.Background())
		}).To(Panic())
	})

	It("panics when host is missing from context", func() {
		Expect(func() {
			sysdig.GetHostFromContext(context.Background())
		}).To(Panic())
	})
})

var _ = Describe("Client authentication", func() {
	var ts *httptest.Server
	var lastHeaders http.Header

	BeforeEach(func() {
		ts = httptest.NewServer(permissionsHandler(func(h http.Header) {
			lastHeaders = h.Clone()
		}))
	})

	AfterEach(func() {
		ts.Close()
	})

	Describe("WithHostAndTokenFromContext", func() {
		It("authenticates using context values", func() {
			client, err := sysdig.NewSysdigClient(sysdig.WithHostAndTokenFromContext())
			Expect(err).NotTo(HaveOccurred())

			ctx := sysdig.WrapContextWithHost(context.Background(), ts.URL)
			ctx = sysdig.WrapContextWithToken(ctx, "ctx-token")

			resp, err := client.GetMyPermissionsWithResponse(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(lastHeaders.Get("Authorization")).To(Equal("Bearer ctx-token"))
		})

		It("fails when token is missing from context", func() {
			client, err := sysdig.NewSysdigClient(sysdig.WithHostAndTokenFromContext())
			Expect(err).NotTo(HaveOccurred())

			ctx := sysdig.WrapContextWithHost(context.Background(), ts.URL)

			_, err = client.GetMyPermissionsWithResponse(ctx)
			Expect(err).To(MatchError(ContainSubstring("authorization token not present")))
		})
	})

	Describe("WithFallbackAuthentication", func() {
		It("uses first auth when it succeeds", func() {
			client, err := sysdig.NewSysdigClient(
				sysdig.WithFallbackAuthentication(
					sysdig.WithFixedHostAndToken(ts.URL, "primary-token"),
					sysdig.WithFixedHostAndToken(ts.URL, "fallback-token"),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.GetMyPermissionsWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(lastHeaders.Get("Authorization")).To(Equal("Bearer primary-token"))
		})

		It("falls back to second auth when first fails", func() {
			client, err := sysdig.NewSysdigClient(
				sysdig.WithFallbackAuthentication(
					sysdig.WithHostAndTokenFromContext(),
					sysdig.WithFixedHostAndToken(ts.URL, "fallback-token"),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.GetMyPermissionsWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(lastHeaders.Get("Authorization")).To(Equal("Bearer fallback-token"))
		})

		It("fails when all auth methods fail", func() {
			client, err := sysdig.NewSysdigClient(
				sysdig.WithFallbackAuthentication(
					sysdig.WithHostAndTokenFromContext(),
					sysdig.WithHostAndTokenFromContext(),
				),
			)
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetMyPermissionsWithResponse(context.Background())
			Expect(err).To(MatchError(ContainSubstring("unable to authenticate")))
		})
	})

	Describe("WithVersion", func() {
		It("sends User-Agent header with version", func() {
			client, err := sysdig.NewSysdigClient(
				sysdig.WithFixedHostAndToken(ts.URL, "tok"),
				sysdig.WithVersion("2.0.0"),
			)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.GetMyPermissionsWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(lastHeaders.Get("User-Agent")).To(Equal("sysdig-mcp-server/2.0.0"))
		})
	})
})
