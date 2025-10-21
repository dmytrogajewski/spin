package constants

// Common error messages
const (
	ErrMsgInvalidInput     = "invalid input"
	ErrMsgNotFound         = "not found"
	ErrMsgUnauthorized     = "unauthorized"
	ErrMsgForbidden        = "forbidden"
	ErrMsgTimeout          = "timeout"
	ErrMsgInternalError    = "internal error"
	ErrMsgInvalidParameter = "invalid parameter"
	ErrMsgMissingRequired  = "missing required parameter"
	ErrMsgInvalidFormat    = "invalid format"
	ErrMsgOperationFailed  = "operation failed"
)

// Common status messages
const (
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
	StatusError     = "error"
	StatusWarning   = "warning"
	StatusInfo      = "info"
)

// File extensions
const (
	ExtGo     = ".go"
	ExtYaml   = ".yaml"
	ExtJSON   = ".json"
	ExtToml   = ".toml"
	ExtMd     = ".md"
	ExtTxt    = ".txt"
	ExtLog    = ".log"
	ExtConfig = ".config"
)

// Network protocols
const (
	ProtocolHTTP  = "http"
	ProtocolHTTPS = "https"
	ProtocolWS    = "ws"
	ProtocolWSS   = "wss"
	ProtocolTCP   = "tcp"
	ProtocolUDP   = "udp"
)

// Common configuration keys
const (
	ConfigKeyAPIKey     = "api_key"
	ConfigKeyBaseURL    = "base_url"
	ConfigKeyTimeout    = "timeout"
	ConfigKeyMaxRetries = "max_retries"
	ConfigKeyDebug      = "debug"
	ConfigKeyLogLevel   = "log_level"
)

// Tool execution constants
const (
	ToolCallTypeFunction = "function"
	ToolRoleTool         = "tool"
	ToolRoleAssistant    = "assistant"
	ToolRoleUser         = "user"
	ToolRoleSystem       = "system"
)

// Message types
const (
	MessageTypeTurnStart         = "turn_start"
	MessageTypeAssistantDelta    = "assistant_delta"
	MessageTypeToolCallProposed  = "tool_call_proposed"
	MessageTypeToolCallExecuting = "tool_call_executing"
	MessageTypeToolCallResult    = "tool_call_result"
	MessageTypeTurnComplete      = "turn_complete"
	MessageTypeStatusUpdate      = "status_update"
)

// Default values
const (
	DefaultTimeout    = 30
	DefaultMaxRetries = 3
	DefaultBufferSize = 1024
	DefaultPort       = 8080
	DefaultHost       = "localhost"
	DefaultLogLevel   = "info"
)
