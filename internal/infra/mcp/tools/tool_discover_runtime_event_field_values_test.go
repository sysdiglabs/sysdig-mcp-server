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

var _ = Describe("ToolDiscoverRuntimeEventFieldValues", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		mockClock  *mocks_clock.MockClock
		tool       *tools.ToolDiscoverRuntimeEventFieldValues
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
		tool = tools.NewToolDiscoverRuntimeEventFieldValues(mockClient, mockClock)
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

	It("passes field, scope_hours, and filter through to the client", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventFieldValuesWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, params *sysdig.GetEventFieldValuesParams, _ ...sysdig.RequestEditorFn) (*sysdig.GetEventFieldValuesResponse, error) {
				Expect(params.Field).To(Equal("kubernetes.cluster.name"))
				Expect(params.From).To(Equal(int64(946677600000000000)))
				Expect(params.To).To(Equal(int64(946684800000000000)))
				Expect(params.Filter).NotTo(BeNil())
				Expect(*params.Filter).To(Equal(`severity in ("0","1","2","3")`))

				body := map[string]any{
					"data": []any{
						map[string]any{"label": "suggested", "options": []any{"prod-gke", "stage-gke"}},
						map[string]any{"label": "other", "options": []any{}},
					},
				}
				return &sysdig.GetEventFieldValuesResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &body,
				}, nil
			})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "discover_runtime_event_field_values",
				Arguments: map[string]any{
					"field":       "kubernetes.cluster.name",
					"scope_hours": 2,
					"filter_expr": `severity in ("0","1","2","3")`,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeFalse())
	})

	It("does not set a filter when filter_expr is omitted (no baseline applied to value discovery)", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventFieldValuesWithResponse(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, params *sysdig.GetEventFieldValuesParams, _ ...sysdig.RequestEditorFn) (*sysdig.GetEventFieldValuesResponse, error) {
				Expect(params.Field).To(Equal("ruleName"))
				Expect(params.Filter).To(BeNil())

				body := map[string]any{}
				return &sysdig.GetEventFieldValuesResponse{
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &body,
				}, nil
			})

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "discover_runtime_event_field_values",
				Arguments: map[string]any{
					"field": "ruleName",
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeFalse())
	})

	It("returns a tool error when field is missing", func(ctx SpecContext) {
		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "discover_runtime_event_field_values",
				Arguments: map[string]any{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
	})

	It("surfaces a client error as a tool error", func(ctx SpecContext) {
		mockClient.EXPECT().GetEventFieldValuesWithResponse(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "discover_runtime_event_field_values",
				Arguments: map[string]any{
					"field": "ruleName",
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeTrue())
	})
})
