package child

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

func TestCardFromSpec_NameAndDescription(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	card := CardFromSpec(spec)
	require.Equal(t, spec.Name, card.Name)
	require.Equal(t, spec.Description, card.Description)
}

func TestCardFromSpec_SkillsFromAllowlist(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	card := CardFromSpec(spec)
	ids := skillIDs(card)
	require.ElementsMatch(t, spec.AllowedTools, ids)
	require.NotContains(t, ids, "write_file")
}

func TestCardFromSpec_CapabilitiesFromAllowlist(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	card := CardFromSpec(spec)
	require.False(t, card.Capabilities.Streaming)
	require.False(t, card.Capabilities.PushNotifications)
	require.Equal(t, a2a.ProtocolBindingNDJSON, card.SupportedInterfaces[0].ProtocolBinding)
	require.Equal(t, URLStdio, card.SupportedInterfaces[0].URL)
}

func TestCardFromSpec_AllBuiltins(t *testing.T) {
	t.Parallel()

	for _, spec := range subagent.Builtins() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()

			card := CardFromSpec(spec)
			require.Equal(t, spec.Name, card.Name)
			require.ElementsMatch(t, spec.AllowedTools, skillIDs(card))
		})
	}
}

func TestCardAndFrame_SpawnDeniedByDefault(t *testing.T) {
	t.Parallel()

	for _, spec := range subagent.Builtins() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()

			require.NotContains(t, skillIDs(CardFromSpec(spec)), subagent.ToolSpawn)
			require.NotContains(t, NewHarness(spec).Frame().Tools, subagent.ToolSpawn)
		})
	}
}

func skillIDs(card a2a.AgentCard) []string {
	ids := make([]string, 0, len(card.Skills))
	for _, skill := range card.Skills {
		ids = append(ids, skill.ID)
	}

	return ids
}
