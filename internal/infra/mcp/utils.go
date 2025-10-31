package mcp

import "github.com/mark3labs/mcp-go/mcp"

func getOrDefault[T any](opt *T, defaultValue T) T {
	if opt == nil {
		return defaultValue
	}
	return *opt
}

func Examples[T any](examples ...T) mcp.PropertyOption {
	return func(schema map[string]any) {
		schema["exampes"] = examples
	}
}

func toPtr[T any](val T) *T {
	return &val
}
