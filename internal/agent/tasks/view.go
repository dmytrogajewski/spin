package tasks

import (
	"context"
	"strings"
)

const (
	// KindAgent is an A2A registry row.
	KindAgent = "agent"
	// KindShell is a start_process / BackgroundTaskManager row.
	KindShell = "shell"
)

// TypedID prefixes a raw id so agent and shell namespaces cannot collide.
func TypedID(kind, raw string) string {
	return kind + ":" + raw
}

// SplitID returns kind and raw id. Untyped ids have an empty kind.
func SplitID(id string) (kind, raw string) {
	kind, raw, ok := strings.Cut(id, ":")
	if !ok || (kind != KindAgent && kind != KindShell) {
		return "", id
	}

	return kind, raw
}

// Row is one unified-view line: kind plus typed id, spec/command, and state.
type Row struct {
	Kind  string
	ID    string
	Spec  string
	State string
}

// ShellSnapshot is a shell-manager row. The manager stays a separate store.
type ShellSnapshot struct {
	ID      string
	Command string
	State   string
}

// Merge builds a view from both families without merging the stores.
func Merge(agents []Record, shells []ShellSnapshot) []Row {
	if len(agents) == 0 && len(shells) == 0 {
		return nil
	}

	out := make([]Row, 0, len(agents)+len(shells))
	for _, rec := range agents {
		out = append(out, Row{
			Kind: KindAgent, ID: TypedID(KindAgent, rec.ID),
			Spec: rec.Spec, State: rec.State,
		})
	}

	for _, sh := range shells {
		out = append(out, Row{
			Kind: KindShell, ID: TypedID(KindShell, sh.ID),
			Spec: sh.Command, State: sh.State,
		})
	}

	return out
}

// ShellSource is the shell-manager surface used by the view. Not the A2A registry.
type ShellSource interface {
	List(ctx context.Context) []ShellSnapshot
	Kill(ctx context.Context, id string) error
}

// CancelView routes cancel to the matching store. Shell Kill is SIGTERM/SIGKILL.
func CancelView(ctx context.Context, id string, agents *Registry, shells ShellSource) error {
	kind, raw := SplitID(id)
	if kind == KindShell && shells != nil {
		return shells.Kill(ctx, raw)
	}

	if kind == "" && inAgent(agents, raw) && inShell(ctx, shells, raw) {
		return ErrAmbiguous
	}

	if kind == "" && inShell(ctx, shells, raw) && !inAgent(agents, raw) {
		return shells.Kill(ctx, raw)
	}

	if agents != nil && (kind == KindAgent || kind == "") {
		return agents.Cancel(ctx, raw)
	}

	return ErrNotFound
}

func inAgent(agents *Registry, id string) bool {
	if agents == nil {
		return false
	}

	_, err := agents.lookup(id)

	return err == nil
}

func inShell(ctx context.Context, shells ShellSource, id string) bool {
	if shells == nil {
		return false
	}

	for _, row := range shells.List(ctx) {
		if row.ID == id {
			return true
		}
	}

	return false
}
