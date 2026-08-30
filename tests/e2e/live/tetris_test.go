//go:build e2e_ollama

// Package live runs live Ollama agent sessions that write artifacts to the testbed.
// Journey: specs/journeys/JOURNEY-live-ollama-tetris.md
package live

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	classicRunSlug = "tetris-classic"
	hudRunSlug     = "tetris-hud"
	mobileRunSlug  = "tetris-mobile"

	classicPrompt = `Create a complete playable web Tetris game in this directory.

Requirements:
- Ship a self-contained game the user can open in a browser (index.html, plus JS/CSS if needed).
- Standard 10x20 playfield, seven tetrominoes (I, O, T, S, Z, J, L), rotation, gravity, line clears.
- Keyboard controls: left/right move, down soft-drop, up or X rotate, space hard-drop.
- The game must actually run: pieces spawn, collide with the floor and stacked blocks, lines disappear.
- Do not leave placeholders, TODOs, or stub functions.

When finished, index.html must be playable by opening it locally.`

	hudPrompt = `Create a complete playable web Tetris game in this directory with a visible HUD.

Requirements:
- Self-contained browser game (index.html plus JS/CSS if needed).
- Standard Tetris: 10x20 well, seven tetrominoes, rotation, gravity, line clears.
- HUD must show: current score, current level, lines cleared, and a next-piece preview.
- Keyboard: left/right move, down soft-drop, up rotate, space hard-drop, P pause.
- Scoring increases on line clears; level increases every 10 lines and speeds gravity.
- No TODOs or stubs. index.html must be playable locally.`

	mobilePrompt = `Create a complete playable web Tetris game in this directory that works with keyboard and touch.

Requirements:
- Self-contained browser game (index.html plus JS/CSS if needed).
- Standard Tetris mechanics: 10x20 well, seven tetrominoes, rotation, gravity, line clears.
- Keyboard controls AND on-screen buttons for left, right, rotate, soft-drop, hard-drop.
- Layout must remain usable at a 390px-wide viewport (mobile).
- Show score. No TODOs or stubs. index.html must be playable locally.`
)

func TestLiveOllama_TetrisClassic(t *testing.T) {
	runTetrisCase(t, tetrisCase{
		slug:   classicRunSlug,
		prompt: classicPrompt,
		mustContainAny: [][]string{
			{"keydown", "keyup", "keyCode", "event.key", "KeyboardEvent"},
			{"tetromino", "TETROMINO", "SHAPES", "PIECES", "tetrominoes"},
			{"canvas", "getContext", "playfield", "board", "grid"},
		},
	})
}

func TestLiveOllama_TetrisHUD(t *testing.T) {
	runTetrisCase(t, tetrisCase{
		slug:   hudRunSlug,
		prompt: hudPrompt,
		mustContainAny: [][]string{
			{"score", "Score"},
			{"level", "Level"},
			{"next", "preview", "Next"},
			{"keydown", "event.key", "KeyboardEvent"},
		},
	})
}

func TestLiveOllama_TetrisMobile(t *testing.T) {
	runTetrisCase(t, tetrisCase{
		slug:   mobileRunSlug,
		prompt: mobilePrompt,
		mustContainAny: [][]string{
			{"touch", "pointer", "button", "ontouch", "click"},
			{"keydown", "event.key", "KeyboardEvent"},
			{"score", "Score"},
		},
	})
}

type tetrisCase struct {
	slug           string
	prompt         string
	mustContainAny [][]string
}

func runTetrisCase(t *testing.T, tc tetrisCase) {
	t.Helper()
	requireOllama(t)

	workDir := filepath.Join(testbedRoot(), tc.slug)
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		t.Fatalf("create test-run dir: %v", err)
	}

	r := runLiveExec(t, workDir, tc.prompt)
	if r.err != nil {
		t.Fatalf("spin exec failed: %v\nstdout:\n%s\nstderr:\n%s", r.err, r.stdout, r.stderr)
	}

	indexPath := filepath.Join(workDir, indexHTMLName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("expected %s in %s: %v\nstdout:\n%s\nstderr:\n%s", indexHTMLName, workDir, err, r.stdout, r.stderr)
	}

	if len(data) < minIndexBytes {
		t.Fatalf("%s is too small (%d bytes)", indexHTMLName, len(data))
	}

	bundle := collectWebSources(t, workDir)
	for _, group := range tc.mustContainAny {
		if !containsAny(bundle, group) {
			t.Errorf("generated sources missing any of %v", group)
		}
	}
}

func collectWebSources(t *testing.T, workDir string) string {
	t.Helper()

	var b strings.Builder

	err := filepath.WalkDir(workDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".html" && ext != ".js" && ext != ".css" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		b.Write(data)
		b.WriteByte('\n')

		return nil
	})
	if err != nil {
		t.Fatalf("walk web sources: %v", err)
	}

	return b.String()
}

func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}

	return false
}
