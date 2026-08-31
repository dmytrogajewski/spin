// Package child serves a local A2A child from a subagent Spec.
package child

import (
	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const (
	tagAllowlist = "allowlist"
	// URLStdio is the default local binding URL on stdio.
	URLStdio       = "stdio://"
	mediaTextPlain = "text/plain"
)

// CardFromSpec builds an Agent Card from a subagent Spec allowlist.
func CardFromSpec(spec *subagent.Spec) a2a.AgentCard {
	return a2a.AgentCard{
		Name:               spec.Name,
		Description:        spec.Description,
		Version:            a2a.ProtocolVersion,
		Skills:             skillsFromAllowlist(spec.AllowedTools),
		DefaultInputModes:  []string{mediaTextPlain},
		DefaultOutputModes: []string{mediaTextPlain},
		SupportedInterfaces: []a2a.AgentInterface{{
			URL:             URLStdio,
			ProtocolBinding: a2a.ProtocolBindingNDJSON,
			ProtocolVersion: a2a.ProtocolVersion,
		}},
	}
}

func skillsFromAllowlist(tools []string) []a2a.AgentSkill {
	skills := make([]a2a.AgentSkill, 0, len(tools))
	for _, tool := range tools {
		skills = append(skills, a2a.AgentSkill{ID: tool, Name: tool, Tags: []string{tagAllowlist}})
	}

	return skills
}
