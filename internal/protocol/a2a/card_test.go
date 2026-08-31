package a2a

// Journey: specs/journeys/JOURNEY-016-a2a-types-and-local-json-rpc-codec.md.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypes_JSONRoundTripCamelCase(t *testing.T) {
	t.Parallel()

	card := fixtureCard()
	task := Task{
		ID:        "task-1",
		ContextID: "ctx-1",
		Status:    TaskStatus{State: TaskStateCompleted},
		History: []Message{{
			MessageID: "msg-1",
			Role:      RoleUser,
			Parts:     []Part{{Text: "hello", MediaType: mediaTextPlain}},
		}},
		Artifacts: []Artifact{{
			ArtifactID: "artifact-1",
			Name:       "result",
			Parts:      []Part{{Text: "hello", MediaType: mediaTextPlain}},
		}},
	}

	cardRaw, err := json.Marshal(card)
	require.NoError(t, err)
	require.Contains(t, string(cardRaw), `"supportedInterfaces"`)
	require.Contains(t, string(cardRaw), `"defaultInputModes"`)
	require.Contains(t, string(cardRaw), `"protocolBinding"`)

	taskRaw, err := json.Marshal(task)
	require.NoError(t, err)
	require.Contains(t, string(taskRaw), `"contextId"`)
	require.Contains(t, string(taskRaw), `"messageId"`)
	require.Contains(t, string(taskRaw), `"artifactId"`)
	require.Contains(t, string(taskRaw), `"TASK_STATE_COMPLETED"`)
	require.Contains(t, string(taskRaw), `"ROLE_USER"`)

	var got Task
	require.NoError(t, json.Unmarshal(taskRaw, &got))
	require.Equal(t, task.ID, got.ID)
	require.Equal(t, RoleUser, got.History[0].Role)
	require.Equal(t, "hello", got.Artifacts[0].Parts[0].Text)
}

func TestTaskState_Terminal(t *testing.T) {
	t.Parallel()

	require.True(t, TaskStateCompleted.Terminal())
	require.True(t, TaskStateFailed.Terminal())
	require.True(t, TaskStateCanceled.Terminal())
	require.True(t, TaskStateRejected.Terminal())
	require.False(t, TaskStateWorking.Terminal())
	require.False(t, TaskStateSubmitted.Terminal())
	require.False(t, TaskStateUnspecified.Terminal())
	require.False(t, TaskStateInputRequired.Terminal())
	require.False(t, TaskStateAuthRequired.Terminal())
	require.False(t, TaskState("nope").Terminal())
}

func TestRoleConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, "ROLE_UNSPECIFIED", string(RoleUnspecified))
	require.Equal(t, "ROLE_USER", string(RoleUser))
	require.Equal(t, "ROLE_AGENT", string(RoleAgent))
}

func TestA2ADomainErrorCodes(t *testing.T) {
	t.Parallel()

	codes := []int{
		CodeTaskNotFound,
		CodeTaskNotCancelable,
		CodePushNotificationNotSupported,
		CodeUnsupportedOperation,
		CodeContentTypeNotSupported,
		CodeInvalidAgentResponse,
		CodeExtendedAgentCardNotConfigured,
		CodeExtensionSupportRequired,
		CodeVersionNotSupported,
	}

	for _, code := range codes {
		require.GreaterOrEqual(t, code, -32009)
		require.LessOrEqual(t, code, -32001)
	}
}

func fixtureCard() AgentCard {
	return AgentCard{
		Name:        "fixture-agent",
		Description: "in-memory A2A fixture",
		Version:     "1.0.0",
		Capabilities: AgentCapabilities{
			ExtendedAgentCard: true,
		},
		DefaultInputModes:  []string{mediaTextPlain},
		DefaultOutputModes: []string{mediaTextPlain},
		SupportedInterfaces: []AgentInterface{{
			URL:             "pipe://local",
			ProtocolBinding: ProtocolBindingNDJSON,
			ProtocolVersion: ProtocolVersion,
		}},
		Skills: []AgentSkill{{
			ID:          "echo",
			Name:        "echo",
			Description: "echo text",
			Tags:        []string{"test"},
		}},
	}
}
