// Package frame defines the per-turn TaskFrame injected into Composer.
package frame

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/agent"
)

const (
	// MaxRenderedBytes caps fixture render so the frame cannot duplicate AGENTS.md.
	MaxRenderedBytes = 2048
	// MaxFieldBytes caps a single short field (objective, output_format).
	MaxFieldBytes = 80
)

const (
	objectiveRegular  = "Complete the user's request."
	objectiveReview   = "Review the change set."
	objectiveCompact  = "Answer the query briefly."
	objectivePlanning = "Produce an implementation plan."
	formatRegular     = "unified diff + test names"
	formatReview      = "bullet findings"
	formatCompact     = "short answer"
	formatPlanning    = "numbered steps"
)

// TaskFrame is the per-turn contract injected into the parent Composer.
type TaskFrame struct {
	Objective       string   `json:"objective"`
	Phase           string   `json:"phase"`
	OutputFormat    string   `json:"output_format"`
	Tools           []string `json:"tools"`
	Sources         []string `json:"sources"`
	Boundaries      []string `json:"boundaries"`
	SuccessCriteria []string `json:"success_criteria"`
}

// FromMode builds a TaskFrame whose phase matches the /mode value.
func FromMode(mode string) TaskFrame {
	phase := PhaseForMode(mode)

	return TaskFrame{
		Objective:       objectiveFor(phase),
		Phase:           phase,
		OutputFormat:    formatFor(phase),
		Tools:           []string{},
		Sources:         []string{},
		Boundaries:      []string{},
		SuccessCriteria: []string{},
	}
}

func objectiveFor(phase string) string {
	switch phase {
	case agent.ModeReview:
		return objectiveReview
	case agent.ModeCompact:
		return objectiveCompact
	case agent.ModePlanning:
		return objectivePlanning
	default:
		return objectiveRegular
	}
}

func formatFor(phase string) string {
	switch phase {
	case agent.ModeReview:
		return formatReview
	case agent.ModeCompact:
		return formatCompact
	case agent.ModePlanning:
		return formatPlanning
	default:
		return formatRegular
	}
}

// MarshalStable returns deterministic JSON suitable as a child spawn payload.
func (f TaskFrame) MarshalStable() ([]byte, error) {
	data, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("marshal task frame: %w", err)
	}

	return data, nil
}

// Render returns the stable JSON text form used in Composer.
func (f TaskFrame) Render() string {
	data, err := f.MarshalStable()
	if err != nil {
		return ""
	}

	return string(data)
}

// WithSources stores path/query strings only; values with newlines are dropped.
func (f TaskFrame) WithSources(sources ...string) TaskFrame {
	kept := make([]string, 0, len(sources))
	for _, src := range sources {
		if src == "" || strings.ContainsAny(src, "\n\r") {
			continue
		}

		kept = append(kept, src)
	}

	f.Sources = kept

	return f
}

// PhaseForMode maps a /mode value onto a TaskFrame phase.
func PhaseForMode(mode string) string {
	switch mode {
	case agent.ModeRegular, agent.ModeReview, agent.ModeCompact, agent.ModePlanning:
		return mode
	default:
		return agent.ModeRegular
	}
}
