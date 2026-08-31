package a2a

import "fmt"

// Standard JSON-RPC 2.0 error codes.
const (
	// CodeParseError is returned when the payload is not JSON (−32700).
	CodeParseError = -32700
	// CodeInvalidRequest is returned when the object is not a JSON-RPC 2.0 request (−32600).
	CodeInvalidRequest = -32600
	// CodeMethodNotFound is returned when the method is unknown (−32601).
	CodeMethodNotFound = -32601
	// CodeInvalidParams is returned when params fail validation (−32602).
	CodeInvalidParams = -32602
	// CodeInternalError is returned for unexpected server failures (−32603).
	CodeInternalError = -32603
)

// A2A domain error codes (−32001..−32009) from A2A 1.0 §5.4.
const (
	// CodeTaskNotFound is TaskNotFoundError (−32001).
	CodeTaskNotFound = -32001
	// CodeTaskNotCancelable is TaskNotCancelableError (−32002).
	CodeTaskNotCancelable = -32002
	// CodePushNotificationNotSupported is PushNotificationNotSupportedError (−32003).
	CodePushNotificationNotSupported = -32003
	// CodeUnsupportedOperation is UnsupportedOperationError (−32004).
	CodeUnsupportedOperation = -32004
	// CodeContentTypeNotSupported is ContentTypeNotSupportedError (−32005).
	CodeContentTypeNotSupported = -32005
	// CodeInvalidAgentResponse is InvalidAgentResponseError (−32006).
	CodeInvalidAgentResponse = -32006
	// CodeExtendedAgentCardNotConfigured is ExtendedAgentCardNotConfiguredError (−32007).
	CodeExtendedAgentCardNotConfigured = -32007
	// CodeExtensionSupportRequired is ExtensionSupportRequiredError (−32008).
	CodeExtensionSupportRequired = -32008
	// CodeVersionNotSupported is VersionNotSupportedError (−32009).
	CodeVersionNotSupported = -32009
)

const (
	msgParseError           = "Invalid JSON payload"
	msgInvalidRequest       = "Request payload validation error"
	msgMethodNotFound       = "Method not found"
	msgInvalidParams        = "Invalid parameters"
	msgInternalError        = "Internal error"
	msgTaskNotFound         = "Task not found"
	msgTaskNotCancelable    = "Task not cancelable"
	msgUnsupportedOperation = "Unsupported operation"
	msgInvalidAgentResponse = "Invalid agent response"
)

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message"`
}

// NewRPCError builds a JSON-RPC error with the given code and message.
func NewRPCError(code int, message string) *RPCError {
	return &RPCError{Code: code, Message: message}
}

// Error implements the error interface.
func (rpcErr *RPCError) Error() string {
	if rpcErr == nil {
		return ""
	}

	return fmt.Sprintf("jsonrpc error %d: %s", rpcErr.Code, rpcErr.Message)
}
