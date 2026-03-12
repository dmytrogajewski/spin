package tools

import (
	"context"
	"testing"
)

// toolErrorCase describes a test case where a tool should return an error result.
type toolErrorCase struct {
	name   string
	params map[string]any
}

// runToolErrorTests runs error case tests for any Tool implementation.
func runToolErrorTests(t *testing.T, tool Tool, cases []toolErrorCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := FromMap(tt.params)

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Success {
				t.Error("expected failure result")
			}
		})
	}
}
