package tools

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/clock"
	"github.com/sysdiglabs/sysdig-mcp-server/internal/infra/sysdig"
)

func Examples[T any](examples ...T) mcp.PropertyOption {
	return func(schema map[string]any) {
		schema["examples"] = examples
	}
}

const requiredPermissionsMetaField = "requiredPermissions"

func WithRequiredPermissions(permissions ...string) mcp.ToolOption {
	return func(t *mcp.Tool) {
		if t.Meta == nil {
			t.Meta = &mcp.Meta{}
		}

		if t.Meta.AdditionalFields == nil {
			t.Meta.AdditionalFields = map[string]any{}
		}

		t.Meta.AdditionalFields[requiredPermissionsMetaField] = permissions
	}
}

func RequiredPermissionsFromTool(tool mcp.Tool) []string {
	if tool.Meta == nil {
		slog.Error("tool does not have a meta field, skipping it just in case", "name", tool.Name)
		return []string{"FAKE-PERMISSION-TO-FORCE-TOOL-TO-NOT-BE-USED"}
	}

	if tool.Meta.AdditionalFields == nil {
		slog.Error("tool does not have additional fields, skipping it just in case", "name", tool.Name)
		return []string{"FAKE-PERMISSION-TO-FORCE-TOOL-TO-NOT-BE-USED"}
	}

	requiredPermissionsAny, requiresPermissions := tool.Meta.AdditionalFields[requiredPermissionsMetaField]
	if !requiresPermissions {
		return nil // no permissions required
	}

	requiredPermissions, isSlice := requiredPermissionsAny.([]string)
	if !isSlice {
		slog.Error("required permissions is not a slice, skipping it just in case, reconfigure the tool properly", "name", tool.Name)
		return []string{"FAKE-PERMISSION-TO-FORCE-TOOL-TO-NOT-BE-USED"}
	}

	return requiredPermissions
}

// --- Time-window support for k8s_list_* Monitor tools --------------------

const (
	windowedQueryTimeout  = "60s"
	timeParamStart        = "start"
	timeParamEnd          = "end"
	startParamDescription = "Start of the query window as an RFC3339 timestamp (e.g. 2026-04-01T00:00:00Z). When omitted, the tool returns an instant snapshot (current behavior). When provided without end, end defaults to now."
	endParamDescription   = "End of the query window as an RFC3339 timestamp (e.g. 2026-04-01T01:00:00Z). Requires start. If in the future, clamped to now."
)

// TimeWindow is a resolved, validated [Start, End] pair for a historical PromQL query.
// A zero-value TimeWindow means no window was requested — the caller should emit its
// existing instant query and leave GetQueryV1Params.Time nil.
type TimeWindow struct {
	Start time.Time
	End   time.Time
}

// IsZero reports whether no time window was requested.
func (w TimeWindow) IsZero() bool {
	return w.Start.IsZero() && w.End.IsZero()
}

// RangeSelector returns the PromQL range-selector literal for this window, e.g. "[3600s]".
func (w TimeWindow) RangeSelector() string {
	return fmt.Sprintf("[%ds]", int64(w.End.Sub(w.Start).Seconds()))
}

// WindowSeconds returns the duration of the window in whole seconds.
func (w TimeWindow) WindowSeconds() int64 {
	return int64(w.End.Sub(w.Start).Seconds())
}

// EvalTime returns a *sysdig.Time suitable for GetQueryV1Params.Time. The value is
// the End instant as unix seconds — the native format accepted by Sysdig's internal
// PromQL stack (confirmed against backend PrometheusFacadeController.java:113).
func (w TimeWindow) EvalTime() (*sysdig.Time, error) {
	if w.IsZero() {
		return nil, nil
	}
	var qt sysdig.Time
	if err := qt.FromQueryTime1(w.End.Unix()); err != nil {
		return nil, fmt.Errorf("building eval time: %w", err)
	}
	return &qt, nil
}

// ApplyToParams sets Time and, for windowed queries, Timeout on params.
// It consolidates the three-step boilerplate (EvalTime, set Time, set Timeout)
// that every k8s_list_* handler would otherwise repeat.
func (w TimeWindow) ApplyToParams(params *sysdig.GetQueryV1Params) error {
	evalTime, err := w.EvalTime()
	if err != nil {
		return err
	}
	params.Time = evalTime
	if !w.IsZero() {
		timeout := sysdig.Timeout(windowedQueryTimeout)
		params.Timeout = &timeout
	}
	return nil
}

// WithTimeWindowParams returns a ToolOption that declares the shared "start" and "end"
// RFC3339 parameters on a tool.
func WithTimeWindowParams() mcp.ToolOption {
	return func(t *mcp.Tool) {
		mcp.WithString(timeParamStart, mcp.Description(startParamDescription))(t)
		mcp.WithString(timeParamEnd, mcp.Description(endParamDescription))(t)
	}
}

// ParseTimeWindow reads "start" and "end" from the request, validates them, and returns
// the resolved TimeWindow.
//
//   - Both absent:       returns zero-value TimeWindow, nil error.
//   - end without start: error ("end requires start").
//   - start without end: end = clk.Now().
//   - invalid RFC3339:   error naming the bad field.
//   - end <= start:      error.
func ParseTimeWindow(request mcp.CallToolRequest, clk clock.Clock) (TimeWindow, error) {
	startStr := mcp.ParseString(request, timeParamStart, "")
	endStr := mcp.ParseString(request, timeParamEnd, "")

	if startStr == "" && endStr == "" {
		return TimeWindow{}, nil
	}

	if startStr == "" && endStr != "" {
		return TimeWindow{}, fmt.Errorf("end requires start")
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return TimeWindow{}, fmt.Errorf("invalid start timestamp %q: must be RFC3339 (e.g. 2026-04-01T00:00:00Z)", startStr)
	}

	var end time.Time
	if endStr == "" {
		end = clk.Now().Truncate(time.Second)
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return TimeWindow{}, fmt.Errorf("invalid end timestamp %q: must be RFC3339 (e.g. 2026-04-01T01:00:00Z)", endStr)
		}
		now := clk.Now().Truncate(time.Second)
		if end.After(now) {
			end = now
		}
	}

	if !end.After(start) {
		return TimeWindow{}, fmt.Errorf("end (%s) must be after start (%s)", end.Format(time.RFC3339), start.Format(time.RFC3339))
	}

	return TimeWindow{Start: start, End: end}, nil
}

