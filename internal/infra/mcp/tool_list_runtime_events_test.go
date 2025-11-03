package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	mocks_clock "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock/mocks"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("ToolListRuntimeEvents", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		mockClock  *mocks_clock.MockClock
		tool       *ToolListRuntimeEvents
		ctrl       *gomock.Controller
		handler    *Handler
		mcpClient  *client.Client
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		mockClock = mocks_clock.NewMockClock(ctrl)
		mockClock.EXPECT().Now().AnyTimes().Return(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
		tool = NewToolListRuntimeEvents(mockClient, mockClock)
		handler = NewHandlerWithTools(tool)

		var err error
		mcpClient, err = handler.ServeInProcessClient()
		Expect(err).NotTo(HaveOccurred())

		_, err = mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should handle a successful request and convert params correctly", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventsV1WithResponse(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, params *sysdig.GetEventsV1Params, _ ...sysdig.RequestEditorFn) (*sysdig.GetEventsV1Response, error) {
			Expect(*params.Limit).To(Equal(int32(10)))
			Expect(*params.Filter).To(ContainSubstring("severity = 4"))
			Expect(*params.To).To(Equal(int64(946684800000000000)))
			Expect(*params.From).To(Equal(int64(946677600000000000)))

			return &sysdig.GetEventsV1Response{
				HTTPResponse: &http.Response{
					StatusCode: 200,
				},
				JSON200: &sysdig.ListEventsResponse{
					Data: []sysdig.Event{},
				},
			}, nil
		})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "list_runtime_events",
				Arguments: map[string]interface{}{
					"limit":       10,
					"scope_hours": 2,
					"filter_expr": "severity = 4",
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())
	})

	It("should use default values when no params are provided", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventsV1WithResponse(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, params *sysdig.GetEventsV1Params, _ ...sysdig.RequestEditorFn) (*sysdig.GetEventsV1Response, error) {
			Expect(*params.Limit).To(Equal(int32(50)))
			Expect(*params.Filter).NotTo(ContainSubstring("severity = 4"))
			Expect(*params.To).To(Equal(int64(946684800000000000)))
			Expect(*params.From).To(Equal(int64(946681200000000000)))

			return &sysdig.GetEventsV1Response{
				HTTPResponse: &http.Response{
					StatusCode: 200,
				},
				JSON200: &sysdig.ListEventsResponse{
					Data: []sysdig.Event{},
				},
			}, nil
		})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "list_runtime_events",
				Arguments: map[string]interface{}{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())
	})

	It("should handle a client error", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventsV1WithResponse(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "list_runtime_events",
				Arguments: map[string]interface{}{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a non-200 status code", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventsV1WithResponse(gomock.Any(), gomock.Any()).Return(&sysdig.GetEventsV1Response{
			HTTPResponse: &http.Response{
				StatusCode: 401,
			},
			Body: []byte("Unauthorized"),
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "list_runtime_events",
				Arguments: map[string]interface{}{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})
})
