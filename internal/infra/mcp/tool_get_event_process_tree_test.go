package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("ToolGetEventProcessTree", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		tool       *ToolGetEventProcessTree
		ctrl       *gomock.Controller
		handler    *Handler
		mcpClient  *client.Client
		checker    PermissionChecker
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		checker = NewPermissionChecker(mockClient)
		mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
			HTTPResponse: &http.Response{
				StatusCode: 200,
			},
			JSON200: &sysdig.UserPermissions{
				Permissions: []string{"policy-events.read"},
			},
		}, nil)
		tool = NewToolGetEventProcessTree(mockClient, checker)
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

	It("should handle a successful request", func(ctx SpecContext) {
		eventID := "12345"
		branchesData := map[string]any{"branches": "data"}
		treesData := map[string]any{"tree": "data"}

		branchesResponse := &sysdig.GetProcessTreeBranchesResponse{
			JSON200: &branchesData,
		}
		treesResponse := &sysdig.GetProcessTreeTreesResponse{
			JSON200: &treesData,
		}

		mockClient.EXPECT().GetProcessTreeBranchesWithResponse(gomock.Any(), eventID).Return(branchesResponse, nil)
		mockClient.EXPECT().GetProcessTreeTreesWithResponse(gomock.Any(), eventID).Return(treesResponse, nil)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_process_tree",
				Arguments: map[string]interface{}{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())

		Expect(result.Content).To(HaveLen(1))
		contentItem := result.Content[0]

		textContent, ok := mcp.AsTextContent(contentItem)
		Expect(ok).To(BeTrue())

		var resultMap map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		Expect(err).NotTo(HaveOccurred())

		Expect(resultMap).To(HaveKeyWithValue("branches", branchesData))
		Expect(resultMap).To(HaveKeyWithValue("tree", treesData))
	})

	It("should return an error if event_id is missing", func(ctx SpecContext) {
		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_event_process_tree",
				Arguments: map[string]interface{}{},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a not found error", func(ctx SpecContext) {
		eventID := "12345"
		mockClient.EXPECT().GetProcessTreeBranchesWithResponse(gomock.Any(), eventID).Return(nil, sysdig.ErrNotFound)
		mockClient.EXPECT().GetProcessTreeTreesWithResponse(gomock.Any(), eventID).Return(nil, sysdig.ErrNotFound)

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_process_tree",
				Arguments: map[string]interface{}{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.IsError).To(BeFalse())

		Expect(result.Content).To(HaveLen(1))
		contentItem := result.Content[0]

		textContent, ok := mcp.AsTextContent(contentItem)
		Expect(ok).To(BeTrue())

		var resultMap map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		Expect(err).NotTo(HaveOccurred())

		Expect(resultMap["metadata"]).To(HaveKeyWithValue("note", "Process tree not available for this event"))
	})

	It("should handle a client error on branches request", func(ctx SpecContext) {
		eventID := "12345"
		mockClient.EXPECT().GetProcessTreeBranchesWithResponse(gomock.Any(), eventID).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_process_tree",
				Arguments: map[string]interface{}{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})

	It("should handle a client error on trees request", func(ctx SpecContext) {
		eventID := "12345"
		branchesData := map[string]any{"branches": "data"}
		branchesResponse := &sysdig.GetProcessTreeBranchesResponse{
			JSON200: &branchesData,
		}
		mockClient.EXPECT().GetProcessTreeBranchesWithResponse(gomock.Any(), eventID).Return(branchesResponse, nil)
		mockClient.EXPECT().GetProcessTreeTreesWithResponse(gomock.Any(), eventID).Return(nil, fmt.Errorf("client error"))

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_event_process_tree",
				Arguments: map[string]interface{}{
					"event_id": eventID,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.IsError).To(BeTrue())
	})
})
