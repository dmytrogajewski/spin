package suggest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	pasteRelDir = ".spin/paste"
	pngExt      = ".png"
	jpgExt      = ".jpg"
)

var (
	pngMagic  = []byte{0x89, 0x50, 0x4e, 0x47}
	jpegMagic = []byte{0xff, 0xd8, 0xff}
)

// PasteResult is text to insert after classifying a paste.
type PasteResult struct {
	Text string
}

// ClassifyPaste turns clipboard/paste bytes into prompt text.
func ClassifyPaste(raw []byte, workDir string) PasteResult {
	if len(raw) == 0 {
		return PasteResult{}
	}

	if ext, ok := imageExt(raw); ok {
		return pasteImage(raw, workDir, ext)
	}

	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	text = strings.TrimRightFunc(text, unicode.IsSpace)

	if paths, ok := existingPastePaths(text, workDir); ok {
		return PasteResult{Text: strings.Join(paths, " ")}
	}

	return PasteResult{Text: string(raw)}
}

func imageExt(raw []byte) (string, bool) {
	if bytes.HasPrefix(raw, pngMagic) {
		return pngExt, true
	}

	if bytes.HasPrefix(raw, jpegMagic) {
		return jpgExt, true
	}

	return "", false
}

func pasteImage(raw []byte, workDir, ext string) PasteResult {
	rel, err := writePasteFile(workDir, raw, ext)
	if err != nil {
		return PasteResult{}
	}

	return PasteResult{Text: "@" + rel}
}

func writePasteFile(workDir string, raw []byte, ext string) (string, error) {
	sum := sha256.Sum256(raw)
	name := hex.EncodeToString(sum[:8]) + ext
	rel := filepath.ToSlash(filepath.Join(pasteRelDir, name))

	abs, ok := containedFile(workDir, rel)
	if !ok {
		return "", os.ErrInvalid
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", fmt.Errorf("paste dir: %w", err)
	}

	if err := os.WriteFile(abs, raw, 0o600); err != nil {
		return "", fmt.Errorf("paste file: %w", err)
	}

	return rel, nil
}

func existingPastePaths(text, workDir string) ([]string, bool) {
	lines := splitPasteLines(text)
	if len(lines) == 0 {
		return nil, false
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		rel, ok := resolvePastePath(line, workDir)
		if !ok {
			return nil, false
		}

		out = append(out, "@"+rel)
	}

	return out, true
}

func splitPasteLines(text string) []string {
	raw := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(raw))

	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}

	return out
}

func resolvePastePath(line, workDir string) (string, bool) {
	line = strings.TrimPrefix(line, "file://")
	if line == "" {
		return "", false
	}

	abs := line
	if !filepath.IsAbs(line) {
		abs = filepath.Join(workDir, line)
	}

	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}

	root, err := filepath.Abs(workDir)
	if err != nil {
		return "", false
	}

	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", false
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}

	return filepath.ToSlash(rel), true
}
