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

const (
	windowedQueryTimeout  = "60s"
	timeParamStart        = "start"
	timeParamEnd          = "end"
	startParamDescription = "Start of the query window as an RFC3339 timestamp (e.g. 2026-04-01T00:00:00Z). When omitted, the tool returns an instant snapshot (current behavior). When provided without end, end defaults to now."
	endParamDescription   = "End of the query window as an RFC3339 timestamp (e.g. 2026-04-01T01:00:00Z). Requires start. If in the future, clamped to now."
)

type TimeWindow struct {
	Start time.Time
	End   time.Time
}

func (w TimeWindow) IsZero() bool {
	return w.Start.IsZero() && w.End.IsZero()
}

func (w TimeWindow) RangeSelector() string {
	return fmt.Sprintf("[%ds]", int64(w.End.Sub(w.Start).Seconds()))
}

func (w TimeWindow) WindowSeconds() int64 {
	return int64(w.End.Sub(w.Start).Seconds())
}

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

func WithTimeWindowParams() mcp.ToolOption {
	return func(t *mcp.Tool) {
		mcp.WithString(timeParamStart, mcp.Description(startParamDescription))(t)
		mcp.WithString(timeParamEnd, mcp.Description(endParamDescription))(t)
	}
}

// Reads "start" and "end" from the request, validates them, and return the resolved TimeWindow.

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

