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

	BeforeEach(func() {
		sysdigUrl := os.Getenv("SYSDIG_MCP_API_HOST")
		sysdigToken := os.Getenv("SYSDIG_MCP_API_SECURE_TOKEN")

		var err error
		client, err = sysdig.NewSysdigClient(sysdigUrl, sysdigToken)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when fetching user permissions", func() {
		It("should return the user permissions successfully", func() {
			resp, err := client.GetMyPermissionsWithResponse(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.JSON200).ToNot(BeNil())
			Expect(resp.JSON200.Permissions).ToNot(BeEmpty())
		})
	})
})
