package tools_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	inframcp "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("ToolGetEventInfo", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		tool       *tools.ToolGetEventInfo
		ctrl       *gomock.Controller
		handler    *inframcp.Handler
		mcpClient  *client.Client
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.UserPermissions{
				Permissions: []string{"policy-events.read"},
			},
		}, nil).AnyTimes()
		tool = tools.NewToolGetEventInfo(mockClient)
		handler = inframcp.NewHandler("dev", mockClient)
		handler.RegisterTools(tool)

		var err error
		mcpClient, err = handler.ServeInProcessClient()
		Expect(err).NotTo(HaveOccurred())

		_, err = mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should handle a successful request", func(ctx SpecContext) {
		eventID := "12345"
		mockClient.EXPECT().GetEventV1WithResponse(gomock.Any(), eventID).Return(&sysdig.GetEventV1Response{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.Event{
				Id: eventID,
			},
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_info",
				Arguments: map[string]any{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())
	})

	It("should return an error if event_id is missing", func(ctx SpecContext) {
		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_event_info",
				Arguments: map[string]any{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a client error", func(ctx SpecContext) {
		eventID := "12345"
		mockClient.EXPECT().GetEventV1WithResponse(gomock.Any(), eventID).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_info",
				Arguments: map[string]any{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a non-200 status code", func(ctx SpecContext) {
		eventID := "12345"
		mockClient.EXPECT().GetEventV1WithResponse(gomock.Any(), eventID).Return(&sysdig.GetEventV1Response{
			HTTPResponse: &http.Response{
				StatusCode: 404,
			},
			Body: []byte("Not Found"),
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_info",
				Arguments: map[string]any{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})
})
