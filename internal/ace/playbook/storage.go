package playbook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/events"
)

// playbookJSON is the JSON representation of a playbook.
type playbookJSON struct {
	Bullets []*bullet.Bullet `json:"bullets"`
}

// Save serializes the playbook to a JSON file.
// Uses atomic writes (temp file + rename) for crash safety.
func (p *Playbook) Save(path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Collect all bullets
	bullets := make([]*bullet.Bullet, 0, len(p.bullets))
	for _, b := range p.bullets {
		bullets = append(bullets, b)
	}

	data := playbookJSON{Bullets: bullets}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal playbook: %w", err)
	}

	// Atomic write: temp file + rename
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "playbook-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write data
	if _, err := tmpFile.Write(jsonData); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Load deserializes a playbook from a JSON file.
// Validates all bullets on load.
func Load(path string, emitter *events.EventEmitter, embedder embedding.Embedder) (*Playbook, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON
	var playbookData playbookJSON
	if err := json.Unmarshal(data, &playbookData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Create playbook
	pb := New(emitter, embedder)

	// Add bullets (validates each one)
	for _, b := range playbookData.Bullets {
		if b == nil {
			continue
		}
		pb.bullets[b.ID] = b
	}

	return pb, nil
}
