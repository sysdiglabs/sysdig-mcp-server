package tools_test

import (
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	mocks_clock "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock/mocks"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/mcp/tools"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
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
		now = time.Date(2026, time.April, 16, 12, 0, 0, 500000000, time.UTC)
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

	It("rejects start at or after the truncated current second (end is not after start)", func() {
		mockClock.EXPECT().Now().Return(now)
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-04-16T12:00:00Z",
		}), mockClock)
		Expect(err).To(MatchError(ContainSubstring("must be after start")))
	})

	It("rejects windows longer than 90 days", func() {
		mockClock.EXPECT().Now().Return(now)
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-01-01T00:00:00Z",
			"end":   "2026-04-30T00:00:00Z",
		}), mockClock)
		Expect(err).To(MatchError(ContainSubstring("exceeds the maximum of 90 days")))
	})

	It("accepts windows of exactly 90 days", func() {
		mockClock.EXPECT().Now().Return(now)
		_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
			"start": "2026-01-01T00:00:00Z",
			"end":   "2026-04-01T00:00:00Z",
		}), mockClock)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("TimeWindow methods", func() {
	var (
		start = time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
		end   = time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)
		tw    = tools.TimeWindow{Start: start, End: end}
	)

	Describe("RangeSelector", func() {
		It("returns the PromQL range-selector literal in seconds for a 1h window", func() {
			Expect(tw.RangeSelector()).To(Equal("[3600s]"))
		})

		It("panics when called on a zero TimeWindow", func() {
			Expect(func() { tools.TimeWindow{}.RangeSelector() }).To(PanicWith("RangeSelector called on zero TimeWindow"))
		})
	})

	Describe("WindowSeconds", func() {
		It("returns the window length in whole seconds for a 1h window", func() {
			Expect(tw.WindowSeconds()).To(Equal(int64(3600)))
		})

		It("panics when called on a zero TimeWindow", func() {
			Expect(func() { tools.TimeWindow{}.WindowSeconds() }).To(PanicWith("WindowSeconds called on zero TimeWindow"))
		})
	})

	Describe("EvalTime", func() {
		It("returns nil eval time for a zero TimeWindow", func() {
			et, err := tools.TimeWindow{}.EvalTime()
			Expect(err).NotTo(HaveOccurred())
			Expect(et).To(BeNil())
		})

		It("returns a sysdig.Time encoding the window's End for a non-zero window", func() {
			et, err := tw.EvalTime()
			Expect(err).NotTo(HaveOccurred())
			Expect(et).NotTo(BeNil())

			var expected sysdig.Time
			Expect(expected.FromQueryTime1(end.Unix())).To(Succeed())
			Expect(*et).To(Equal(expected))
		})
	})

	Describe("ApplyToParams", func() {
		It("leaves params untouched on a zero TimeWindow", func() {
			params := &sysdig.GetQueryV1Params{}
			Expect(tools.TimeWindow{}.ApplyToParams(params)).To(Succeed())
			Expect(params.Time).To(BeNil())
			Expect(params.Timeout).To(BeNil())
		})

		It("sets Time and Timeout for a non-zero window", func() {
			params := &sysdig.GetQueryV1Params{}
			Expect(tw.ApplyToParams(params)).To(Succeed())
			Expect(params.Time).NotTo(BeNil())
			Expect(params.Timeout).NotTo(BeNil())
			Expect(*params.Timeout).To(Equal(sysdig.Timeout("60s")))
		})
	})
})
