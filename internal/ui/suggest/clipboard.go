package suggest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const clipTimeout = 500 * time.Millisecond

// ErrEmptyClipboard is returned when no clipboard tool has data.
var ErrEmptyClipboard = errors.New("empty clipboard")

// Clipper reads clipboard bytes (text or image).
type Clipper func() ([]byte, error)

// OSClipboard tries Wayland then X11 tools, image first then text.
func OSClipboard() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clipTimeout)
	defer cancel()

	if data, err := runClip(ctx, "wl-paste", "--type", "image/png"); err == nil && len(data) > 0 {
		return data, nil
	}

	if data, err := runClip(ctx, "xclip", "-selection", "clipboard", "-t", "image/png", "-o"); err == nil && len(data) > 0 {
		return data, nil
	}

	if data, err := runClip(ctx, "wl-paste", "-n"); err == nil && len(data) > 0 {
		return data, nil
	}

	if data, err := runClip(ctx, "xclip", "-selection", "clipboard", "-o"); err == nil && len(data) > 0 {
		return data, nil
	}

	return nil, ErrEmptyClipboard
}

func runClip(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var buf bytes.Buffer

	cmd.Stdout = &buf
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("clipboard %s: %w", name, err)
	}

	return bytes.TrimRight(buf.Bytes(), "\x00"), nil
}
