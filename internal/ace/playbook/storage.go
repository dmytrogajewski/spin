package playbook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/storage"
)

// CurrentPlaybookVersion is the current playbook schema version.
const CurrentPlaybookVersion = 1

// playbookJSON is the JSON representation of a playbook.
type playbookJSON struct {
	Version int              `json:"version"`
	Bullets []*bullet.Bullet `json:"bullets"`
}

// Save serializes the playbook to a JSON file.
// Uses atomic writes (temp file + rename) for crash safety.
func (p *Playbook) Save(path string) error {
	path = filepath.Clean(path)

	data := playbookJSON{
		Version: CurrentPlaybookVersion,
		Bullets: p.bullets.Values(),
	}

	// Marshal to JSON.
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal playbook: %w", err)
	}

	return storage.AtomicWriteFile(path, jsonData, 0o600)
}

// Load deserializes a playbook from a JSON file.
// Validates all bullets on load.
func Load(path string, emitter *events.EventEmitter, embedder embedding.Embedder) (*Playbook, error) {
	path = filepath.Clean(path)

	// Read file.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON.
	var playbookData playbookJSON

	err = json.Unmarshal(data, &playbookData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Create playbook.
	pb := New(emitter, embedder)

	// Add bullets (validates each one).
	for _, b := range playbookData.Bullets {
		if b == nil {
			continue
		}

		pb.bullets.Set(b.ID, b)
	}

	return pb, nil
}
