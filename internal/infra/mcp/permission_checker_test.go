package mcp_test

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig/mocks"
)

var _ = Describe("CachedPermissionChecker", func() {
	var (
		mockClient *mocks.MockExtendedClientWithResponsesInterface
		checker    mcp.PermissionChecker
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockExtendedClientWithResponsesInterface(ctrl)
		checker = mcp.NewPermissionChecker(mockClient)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("when permissions are loaded successfully", func() {
		BeforeEach(func() {
			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(&sysdig.GetMyPermissionsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &sysdig.UserPermissions{
					Permissions: []string{"policy-events.read", "some-other.permission"},
				},
			}, nil).Times(1)
		})

		It("returns true for a permission the user has", func() {
			Expect(checker.HasPermission("policy-events.read")).To(BeTrue())
		})

		It("returns false for a permission the user does not have", func() {
			Expect(checker.HasPermission("non-existent.permission")).To(BeFalse())
		})

		It("only calls the client once for multiple checks", func() {
			Expect(checker.HasPermission("policy-events.read")).To(BeTrue())
			Expect(checker.HasPermission("some-other.permission")).To(BeTrue())
			Expect(checker.HasPermission("non-existent.permission")).To(BeFalse())
		})
	})

	Context("when the client returns an error", func() {
		BeforeEach(func() {
			mockClient.EXPECT().GetMyPermissionsWithResponse(gomock.Any()).Return(nil, errors.New("API error")).Times(1)
		})

		It("returns false for any permission check", func() {
			Expect(checker.HasPermission("policy-events.read")).To(BeFalse())
		})

		It("only calls the client once for multiple checks", func() {
			Expect(checker.HasPermission("policy-events.read")).To(BeFalse())
			Expect(checker.HasPermission("some-other.permission")).To(BeFalse())
		})
	})
})
