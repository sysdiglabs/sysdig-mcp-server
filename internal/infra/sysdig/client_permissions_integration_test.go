package sysdig_test

import (
	"context"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

var _ = Describe("Sysdig Permissions Client", func() {
	var client sysdig.ExtendedClientWithResponsesInterface
	var sysdigURL string
	var sysdigToken string

	BeforeEach(func() {
		sysdigURL = os.Getenv("SYSDIG_MCP_API_HOST")
		sysdigToken = os.Getenv("SYSDIG_MCP_API_TOKEN")
		if sysdigURL == "" || sysdigToken == "" {
			Skip("requires SYSDIG_MCP_API_HOST and SYSDIG_MCP_API_TOKEN")
		}
	})

	Context("when fetching user permissions", func() {
		It("should return the user permissions successfully", func() {
			var err error
			client, err = sysdig.NewSysdigClient(sysdig.WithFixedHostAndToken(sysdigURL, sysdigToken))
			Expect(err).ToNot(HaveOccurred())

			resp, err := client.GetMyPermissionsWithResponse(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.JSON200).ToNot(BeNil())
			Expect(resp.JSON200.Permissions).ToNot(BeEmpty())
		})
	})
})
