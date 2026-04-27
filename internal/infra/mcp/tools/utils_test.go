package tools_test

import (
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	mocks_clock "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock/mocks"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
)

var _ = Describe("ParseTimeWindow", func() {
	var (
		ctrl      *gomock.Controller
		mockClock *mocks_clock.MockClock
		now       time.Time
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClock = mocks_clock.NewMockClock(ctrl)
		now = time.Date(2026, time.April, 16, 12, 0, 0, 500000000, time.UTC) // has sub-second part
	})

	AfterEach(func() { ctrl.Finish() })

	makeRequest := func(args map[string]any) mcp.CallToolRequest {
		return mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: args},
		}
	}

	It("returns zero TimeWindow when neither start nor end is provided", func() {
		tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{}), mockClock)
		Expect(err).NotTo(HaveOccurred())
		Expect(tw.IsZero()).To(BeTrue())
	})

	It("returns error when end is provided without start", func() {
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"end": "2026-04-16T11:00:00Z",
		}), mockClock)
		Expect(err).To(MatchError(ContainSubstring("end requires start")))
	})

	It("returns error for invalid RFC3339 start", func() {
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "not-a-timestamp",
		}), mockClock)
		Expect(err).To(MatchError(ContainSubstring("invalid start timestamp")))
	})

	It("returns error for invalid RFC3339 end", func() {
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T10:00:00Z",
			"end":   "not-a-timestamp",
		}), mockClock)
		Expect(err).To(MatchError(ContainSubstring("invalid end timestamp")))
	})

	It("clamps end to now when end is in the future", func() {
		mockClock.EXPECT().Now().Return(now)
		tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T10:00:00Z",
			"end":   "2099-01-01T00:00:00Z",
		}), mockClock)
		Expect(err).NotTo(HaveOccurred())
		Expect(tw.End).To(Equal(now.Truncate(time.Second)))
	})

	It("defaults end to now when only start is provided", func() {
		mockClock.EXPECT().Now().Return(now)
		tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T10:00:00Z",
		}), mockClock)
		Expect(err).NotTo(HaveOccurred())
		Expect(tw.End).To(Equal(now.Truncate(time.Second)))
		Expect(tw.Start).To(Equal(time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)))
	})

	It("returns error when end is not after start", func() {
		mockClock.EXPECT().Now().Return(now)
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T11:00:00Z",
			"end":   "2026-04-16T10:00:00Z",
		}), mockClock)
		Expect(err).To(MatchError(ContainSubstring("must be after start")))
	})

	It("returns correct TimeWindow when both start and end are valid past timestamps", func() {
		mockClock.EXPECT().Now().Return(now)
		tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T10:00:00Z",
			"end":   "2026-04-16T11:00:00Z",
		}), mockClock)
		Expect(err).NotTo(HaveOccurred())
		Expect(tw.Start).To(Equal(time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)))
		Expect(tw.End).To(Equal(time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)))
	})

	It("truncates sub-second precision from now so RangeSelector never emits [0s]", func() {
		// now has 500ms; start is in the same second — after truncation end == start,
		// which must be rejected rather than silently producing [0s].
		mockClock.EXPECT().Now().Return(now)
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T12:00:00Z",
		}), mockClock)
		Expect(err).To(HaveOccurred())
	})
})
