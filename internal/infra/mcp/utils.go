package mcp

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
)

func Examples[T any](examples ...T) mcp.PropertyOption {
	return func(schema map[string]any) {
		schema["exampes"] = examples
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

func toPtr[T any](val T) *T {
	return &val
}
