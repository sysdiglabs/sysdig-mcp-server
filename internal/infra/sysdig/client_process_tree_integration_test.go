package sysdig_test

import (
	"context"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

var _ = Describe("Sysdig Process Tree Client", func() {
	var (
		client  sysdig.ExtendedClientWithResponsesInterface
		eventId string
	)

	BeforeEach(func() {
		sysdigUrl := os.Getenv("SYSDIG_MCP_API_HOST")
		sysdigToken := os.Getenv("SYSDIG_MCP_API_SECURE_TOKEN")

		var err error
		client, err = sysdig.NewSysdigClient(sysdigUrl, sysdigToken)
		Expect(err).ToNot(HaveOccurred())

		eventId = "18748b13ef9d1deb89204bbc42d56b7d"
	})

	Context("when fetching the process tree for an event", func() {
		It("should return the process tree branches successfully", func() {
			resp, err := client.GetProcessTreeBranchesWithResponse(context.Background(), eventId)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.JSON200).ToNot(BeNil())
		})

		It("should return the process tree successfully", func() {
			resp, err := client.GetProcessTreeTreesWithResponse(context.Background(), eventId)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.JSON200).ToNot(BeNil())
		})
	})
})
