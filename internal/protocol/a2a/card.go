// Package a2a implements A2A 1.0 types, a local NDJSON binding, and HTTPS JSON-RPC.
package a2a

// ProtocolVersion is the A2A protocol version this package speaks.
const ProtocolVersion = "1.0"

// ProtocolBindingNDJSON is the local custom binding name on AgentInterface.
const ProtocolBindingNDJSON = "NDJSON-RPC"

// AgentCard is the A2A self-describing agent manifest.
type AgentCard struct {
	Capabilities        AgentCapabilities `json:"capabilities"`
	DefaultInputModes   []string          `json:"defaultInputModes"`
	DefaultOutputModes  []string          `json:"defaultOutputModes"`
	Description         string            `json:"description"`
	Name                string            `json:"name"`
	Skills              []AgentSkill      `json:"skills"`
	SupportedInterfaces []AgentInterface  `json:"supportedInterfaces"`
	Version             string            `json:"version"`
}

// AgentCapabilities is the optional A2A capability set.
type AgentCapabilities struct {
	ExtendedAgentCard bool `json:"extendedAgentCard,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
	Streaming         bool `json:"streaming,omitempty"`
}

// AgentInterface declares a URL, binding, and protocol version.
type AgentInterface struct {
	ProtocolBinding string `json:"protocolBinding"`
	ProtocolVersion string `json:"protocolVersion"`
	URL             string `json:"url"`
}

// AgentSkill describes one advertised agent ability.
type AgentSkill struct {
	Description string   `json:"description"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tags        []string `json:"tags"`
}
