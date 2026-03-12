package config

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// FuzzV2_UnmarshalYAML fuzzes the YAML unmarshaling of V2.
// This ensures that arbitrary YAML input doesn't cause panics or unexpected behavior.
func FuzzV2_UnmarshalYAML(f *testing.F) {
	// Add seed corpus with known valid configs.
	f.Add([]byte(`version: "2.0"
llm:
  provider: "ollama"
  model: "qwen"
  temperature: 0.7
  max_tokens: 4096
  timeout: 5m
agent:
  max_turns: 10
  timeout: 60m
  work_dir: "."
`))

	f.Add([]byte(`version: "2.0"
llm:
  provider: "openai"
  model: "gpt-4"
  temperature: 0.8
  max_tokens: 2048
  timeout: 3m
agent:
  max_turns: 20
  timeout: 30m
  work_dir: "/tmp"
ace:
  enabled: true
  playbook_path: "/path/to/playbook.json"
  trajectory_path: "/path/to/trajectories/"
  top_k: 5
  min_score: 0.3
`))

	f.Add([]byte(`{}`))
	f.Add([]byte(`version: "2.0"`))
	f.Add([]byte(`invalid yaml: [unclosed`))

	f.Fuzz(func(_ *testing.T, data []byte) {
		// Fuzzing should not panic.
		var cfg V2

		_ = yaml.Unmarshal(data, &cfg)

		// If unmarshal succeeded, validation might fail (expected)
		// But it should never panic.
		_ = cfg.Validate()
	})
}

// FuzzLoaderV2_LoadFromYAML fuzzes the loader's ability to handle arbitrary YAML.
func FuzzLoaderV2_LoadFromYAML(f *testing.F) {
	// Add seed corpus.
	f.Add([]byte(`version: "2.0"
llm:
  provider: "ollama"
  model: "qwen"
  temperature: 0.7
  max_tokens: 4096
  timeout: 5m
agent:
  max_turns: 10
  timeout: 60m
  work_dir: "."
ace:
  enabled: true
  playbook_path: "~/.spin/playbooks/default.json"
  trajectory_path: "~/.spin/trajectories/"
  top_k: 5
  min_score: 0.3
security:
  sandbox_mode: "none"
protocol:
  enable_git: true
  enable_shell: true
  shell_timeout: 30s
`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create a temp file.
		tmpFile := t.TempDir() + "/config.yaml"
		err := writeFile(tmpFile, data)
		if err != nil {
			return // Skip if we can't write the file.
		}

		// Try to load it - should not panic.
		loader := NewLoaderV2()
		_, _ = loader.LoadFromFile(tmpFile)
		// We don't care if it fails, just that it doesn't panic.
	})
}

// writeFile is a helper for fuzz tests.
func writeFile(path string, data []byte) error {
	err := os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}

	return nil
}
