package sysdig_test

import (
	"context"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

var _ = Describe("Sysdig Generate Sysql Client", func() {
	var client sysdig.ExtendedClientWithResponsesInterface

	BeforeEach(func() {
		sysdigURL := os.Getenv("SYSDIG_MCP_API_HOST")
		sysdigToken := os.Getenv("SYSDIG_MCP_API_SECURE_TOKEN")

		var err error
		client, err = sysdig.NewSysdigClient(sysdigURL, sysdigToken)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when generating a sysql query", func() {
		It("should return the sysql query successfully", func(ctx context.Context) {
			resp, err := client.GenerateSysqlWithResponse(ctx, "what are the latest events?")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.JSON200).ToNot(BeNil())
			Expect(resp.JSON200.Text).ToNot(BeEmpty())
		})
	})
})
