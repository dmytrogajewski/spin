package exec

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/exec/format"
)

func TestNewFormatter_Text(t *testing.T) {
	formatter, err := NewFormatter(format.FormatText)
	if err != nil {
		t.Fatalf("NewFormatter(FormatText) returned error: %v", err)
	}
	if formatter == nil {
		t.Fatal("NewFormatter(FormatText) returned nil formatter")
	}

	// Verify it implements the interface
	var _ format.Formatter = formatter
}

func TestNewFormatter_JSON(t *testing.T) {
	formatter, err := NewFormatter(format.FormatJSON)
	if err != nil {
		t.Fatalf("NewFormatter(FormatJSON) returned error: %v", err)
	}
	if formatter == nil {
		t.Fatal("NewFormatter(FormatJSON) returned nil formatter")
	}

	// Verify it implements the interface
	var _ format.Formatter = formatter
}

func TestNewFormatter_Invalid(t *testing.T) {
	tests := []format.OutputFormat{
		format.OutputFormat("invalid"),
		format.OutputFormat("xml"),
		format.OutputFormat(""),
	}

	for _, f := range tests {
		t.Run(string(f), func(t *testing.T) {
			formatter, err := NewFormatter(f)
			if err == nil {
				t.Errorf("NewFormatter(%q) should return error for invalid format", f)
			}
			if formatter != nil {
				t.Errorf("NewFormatter(%q) should return nil formatter on error", f)
			}
		})
	}
}

func TestNewFormatter_Default(t *testing.T) {
	// Test that text is the recommended default
	formatter, err := NewFormatter(format.FormatText)
	if err != nil {
		t.Fatalf("NewFormatter(FormatText) failed: %v", err)
	}

	// Basic sanity check
	output := formatter.FormatStart("test")
	if output == "" {
		t.Error("Formatter produced empty output")
	}
}
