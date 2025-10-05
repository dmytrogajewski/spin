package exec

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/exec/format"
)

// NewFormatter creates a new formatter for the specified output format.
// Returns an error if the format is not supported.
func NewFormatter(outputFormat format.OutputFormat) (format.Formatter, error) {
	switch outputFormat {
	case format.FormatText:
		return format.NewTextFormatter(), nil
	case format.FormatJSON:
		return format.NewJSONFormatter(), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}
