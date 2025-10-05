package format

import (
	"encoding/json"
)

// JSONFormatter formats output as structured JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// FormatStart formats the initial message as JSON.
func (f *JSONFormatter) FormatStart(prompt string) string {
	output := map[string]interface{}{
		"type":   "start",
		"prompt": prompt,
	}

	data, _ := json.Marshal(output)
	return string(data) + "\n"
}

// FormatDelta formats a streaming chunk as JSON.
func (f *JSONFormatter) FormatDelta(delta string) string {
	if delta == "" {
		return ""
	}

	output := map[string]interface{}{
		"type":    "delta",
		"content": delta,
	}

	data, _ := json.Marshal(output)
	return string(data) + "\n"
}

// FormatComplete formats the completion message as JSON.
func (f *JSONFormatter) FormatComplete(result *ExecResult) string {
	output := map[string]interface{}{
		"status":       result.Status,
		"tokens_used":  result.TokensUsed,
		"duration_ms":  result.Duration.Milliseconds(),
	}

	// Files modified (ensure it's an array, not null)
	if result.FilesModified != nil {
		output["files_modified"] = result.FilesModified
	} else {
		output["files_modified"] = []string{}
	}

	// Commands executed (ensure it's an array, not null)
	if result.CommandsRun != nil {
		commands := make([]map[string]interface{}, len(result.CommandsRun))
		for i, cmd := range result.CommandsRun {
			commands[i] = map[string]interface{}{
				"command":   cmd.Command,
				"exit_code": cmd.ExitCode,
				"output":    cmd.Output,
			}
		}
		output["commands_executed"] = commands
	} else {
		output["commands_executed"] = []map[string]interface{}{}
	}

	// Messages (ensure it's an array, not null)
	if result.Messages != nil {
		messages := make([]map[string]interface{}, len(result.Messages))
		for i, msg := range result.Messages {
			messages[i] = map[string]interface{}{
				"role":      msg.Role,
				"content":   msg.Content,
				"timestamp": msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
		output["messages"] = messages
	} else {
		output["messages"] = []map[string]interface{}{}
	}

	// Error (null if no error)
	if result.Error != nil {
		output["error"] = result.Error.Error()
	} else {
		output["error"] = nil
	}

	data, _ := json.Marshal(output)
	return string(data) + "\n"
}

// FormatError formats an error message as JSON.
func (f *JSONFormatter) FormatError(err error) string {
	output := map[string]interface{}{
		"type": "error",
	}

	if err != nil {
		output["error"] = err.Error()
	} else {
		output["error"] = nil
	}

	data, _ := json.Marshal(output)
	return string(data) + "\n"
}
