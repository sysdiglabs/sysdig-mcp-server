package tools_test

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
	inframcp "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("ToolCountRuntimeEvents", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		mockClock  *mocks_clock.MockClock
		tool       *tools.ToolCountRuntimeEvents
		ctrl       *gomock.Controller
		handler    *inframcp.Handler
		mcpClient  *client.Client
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any(), gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &sysdig.UserPermissions{
				Permissions: []string{"policy-events.read"},
			},
		}, nil).AnyTimes()
		mockClock = mocks_clock.NewMockClock(ctrl)
		mockClock.EXPECT().Now().AnyTimes().Return(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
		tool = tools.NewToolCountRuntimeEvents(mockClient, mockClock)
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

	It("converts a request into count params with baseline filter prepended", func(ctx SpecContext) {
		mockClient.EXPECT().GetSecureEventsCountWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, params *sysdig.GetSecureEventsCountParams, _ ...sysdig.RequestEditorFn) (*sysdig.GetSecureEventsCountResponse, error) {
				Expect(params.From).To(Equal(int64(946677600000000000))) // 2000-01-01 minus 2h
				Expect(params.To).To(Equal(int64(946684800000000000)))   // 2000-01-01 00:00:00 UTC
				Expect(*params.Filter).To(ContainSubstring(`not originator in ("benchmarks","compliance","cloudsec","scanning","hostscanning")`))
				Expect(*params.Filter).To(ContainSubstring(`severity = 4`))

				body := map[string]any{
					"policyEvents": map[string]any{"countBySeverity": map[string]any{"0": 1.0}},
				}
				return &sysdig.GetSecureEventsCountResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &body,
				}, nil
			})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "count_runtime_events",
				Arguments: map[string]any{
					"scope_hours": 2,
					"filter_expr": "severity = 4",
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeFalse())
	})

	It("uses defaults (1h window, baseline filter only) when no args provided", func(ctx SpecContext) {
		mockClient.EXPECT().GetSecureEventsCountWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, params *sysdig.GetSecureEventsCountParams, _ ...sysdig.RequestEditorFn) (*sysdig.GetSecureEventsCountResponse, error) {
				Expect(params.From).To(Equal(int64(946681200000000000))) // 2000-01-01 minus 1h
				Expect(params.To).To(Equal(int64(946684800000000000)))
				Expect(*params.Filter).To(Equal(`not originator in ("benchmarks","compliance","cloudsec","scanning","hostscanning")`))

				body := map[string]any{}
				return &sysdig.GetSecureEventsCountResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &body,
				}, nil
			})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "count_runtime_events",
				Arguments: map[string]any{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeFalse())
	})

	It("surfaces a client error as a tool error", func(ctx SpecContext) {
		mockClient.EXPECT().GetSecureEventsCountWithResponse(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "count_runtime_events",
				Arguments: map[string]any{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
	})

	It("surfaces a non-2xx HTTP response as a tool error", func(ctx SpecContext) {
		mockClient.EXPECT().GetSecureEventsCountWithResponse(gomock.Any(), gomock.Any()).Return(&sysdig.GetSecureEventsCountResponse{
			HTTPResponse: &http.Response{StatusCode: 401},
			Body:         []byte("Unauthorized"),
		}, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "count_runtime_events",
				Arguments: map[string]any{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
	})
})
