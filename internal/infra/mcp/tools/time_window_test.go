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

var _ = Describe("TimeWindow helper", func() {
	var (
		ctrl      *gomock.Controller
		mockClock *mocks_clock.MockClock
		now       time.Time
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClock = mocks_clock.NewMockClock(ctrl)
		now = time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC)
		mockClock.EXPECT().Now().AnyTimes().Return(now)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	makeRequest := func(args map[string]any) mcp.CallToolRequest {
		return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
	}

	Describe("ParseTimeWindow", func() {
		When("neither start nor end is provided", func() {
			It("returns a zero-value TimeWindow and no error", func() {
				tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{}), mockClock)
				Expect(err).NotTo(HaveOccurred())
				Expect(tw.IsZero()).To(BeTrue())
			})

			It("treats empty strings as absent", func() {
				tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{"start": "", "end": ""}), mockClock)
				Expect(err).NotTo(HaveOccurred())
				Expect(tw.IsZero()).To(BeTrue())
			})
		})

		When("only end is provided", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{"end": "2026-04-16T10:00:00Z"}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("end requires start")))
			})
		})

		When("only start is provided", func() {
			It("defaults end to the current clock", func() {
				tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{"start": "2026-04-16T10:00:00Z"}), mockClock)
				Expect(err).NotTo(HaveOccurred())
				Expect(tw.Start).To(Equal(time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)))
				Expect(tw.End).To(Equal(now))
			})
		})

		When("both start and end are provided", func() {
			It("returns the parsed TimeWindow", func() {
				tw, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "2026-04-16T11:00:00Z",
				}), mockClock)
				Expect(err).NotTo(HaveOccurred())
				Expect(tw.Start).To(Equal(time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)))
				Expect(tw.End).To(Equal(time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)))
			})
		})

		When("start is not RFC3339", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{"start": "bogus"}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("invalid start timestamp")))
			})
		})

		When("end is not RFC3339", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "bogus",
				}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("invalid end timestamp")))
			})
		})

		When("end equals start", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "2026-04-16T10:00:00Z",
				}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("must be after start")))
			})
		})

		When("end is before start", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T11:00:00Z",
					"end":   "2026-04-16T10:00:00Z",
				}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("must be after start")))
			})
		})

		When("end is more than 60s in the future", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "2026-04-16T12:02:00Z", // 2m ahead of frozen clock
				}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("must not be more than")))
			})
		})

		When("end is within the 60s grace", func() {
			It("is accepted", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "2026-04-16T12:00:30Z", // 30s ahead
				}), mockClock)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("the window exceeds the default maximum", func() {
			It("returns an error", func() {
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-09T11:59:59Z", // 7d + 1s ago
					"end":   "2026-04-16T12:00:00Z",
				}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("exceeds maximum")))
			})
		})

		When("SYSDIG_MCP_MAX_INTERVAL is set", func() {
			It("honours a smaller override", func() {
				GinkgoT().Setenv("SYSDIG_MCP_MAX_INTERVAL", "1h")
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "2026-04-16T12:00:00Z", // 2h window
				}), mockClock)
				Expect(err).To(MatchError(ContainSubstring("exceeds maximum")))
				Expect(err).To(MatchError(ContainSubstring("SYSDIG_MCP_MAX_INTERVAL")))
			})

			It("honours a larger override", func() {
				GinkgoT().Setenv("SYSDIG_MCP_MAX_INTERVAL", "720h") // 30d
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-01T12:00:00Z", // 15d window
					"end":   "2026-04-16T12:00:00Z",
				}), mockClock)
				Expect(err).NotTo(HaveOccurred())
			})

			It("falls back to the default when the env value is unparseable", func() {
				GinkgoT().Setenv("SYSDIG_MCP_MAX_INTERVAL", "not-a-duration")
				_, err := tools.ParseTimeWindow(makeRequest(map[string]any{
					"start": "2026-04-16T10:00:00Z",
					"end":   "2026-04-16T11:00:00Z",
				}), mockClock)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("TimeWindow.RangeSelector", func() {
		DescribeTable("formats the duration as integer seconds",
			func(start, end time.Time, expected string) {
				tw := tools.TimeWindow{Start: start, End: end}
				Expect(tw.RangeSelector()).To(Equal(expected))
			},
			Entry("1 hour",
				time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC),
				"[3600s]"),
			Entry("30 minutes",
				time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 16, 10, 30, 0, 0, time.UTC),
				"[1800s]"),
			Entry("7 days",
				time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
				"[604800s]"),
		)
	})

	Describe("TimeWindow.EvalTime", func() {
		It("returns nil for a zero TimeWindow", func() {
			tw := tools.TimeWindow{}
			qt, err := tw.EvalTime()
			Expect(err).NotTo(HaveOccurred())
			Expect(qt).To(BeNil())
		})

		It("encodes End as unix seconds (10-digit, not UnixNano)", func() {
			end := time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC)
			tw := tools.TimeWindow{Start: end.Add(-time.Hour), End: end}
			qt, err := tw.EvalTime()
			Expect(err).NotTo(HaveOccurred())
			Expect(qt).NotTo(BeNil())

			asUnixSec, err := qt.AsQueryTime1()
			Expect(err).NotTo(HaveOccurred())
			Expect(asUnixSec).To(Equal(end.Unix()))
			// Sanity check: guards against UnixNano() slipping in.
			Expect(asUnixSec).To(BeNumerically("<", int64(1e12)))
		})
	})

	Describe("TimeWindow.IsZero", func() {
		It("is true for the zero value", func() {
			Expect(tools.TimeWindow{}.IsZero()).To(BeTrue())
		})

		It("is false once Start is set", func() {
			Expect(tools.TimeWindow{Start: time.Now()}.IsZero()).To(BeFalse())
		})

		It("is false once End is set", func() {
			Expect(tools.TimeWindow{End: time.Now()}.IsZero()).To(BeFalse())
		})
	})
})
