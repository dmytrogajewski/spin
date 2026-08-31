package a2a

// JSON-RPC 2.0 method names from the spin A2A SPEC (slash form).
const (
	// MethodMessageSend is message/send.
	MethodMessageSend = "message/send"
	// MethodMessageStream is message/stream.
	MethodMessageStream = "message/stream"
	// MethodTasksGet is tasks/get.
	MethodTasksGet = "tasks/get"
	// MethodTasksList is tasks/list.
	MethodTasksList = "tasks/list"
	// MethodTasksCancel is tasks/cancel.
	MethodTasksCancel = "tasks/cancel"
	// MethodAgentGetCard fetches the (extended) Agent Card.
	MethodAgentGetCard = "agent/getAuthenticatedExtendedCard"
	// MethodAgentCard is the local custom-binding card announce notification.
	MethodAgentCard = "agent/card"
)

// SendMessageParams is the params object for message/send and message/stream.
type SendMessageParams struct {
	Configuration *SendMessageConfiguration `json:"configuration,omitempty"`
	Message       Message                   `json:"message"`
}

// SendMessageConfiguration controls blocking vs immediate return.
type SendMessageConfiguration struct {
	ReturnImmediately bool `json:"returnImmediately,omitempty"`
}

// SendMessageResult is the result object for message/send.
type SendMessageResult struct {
	Task *Task `json:"task,omitempty"`
}

// GetTaskParams is the params object for tasks/get.
type GetTaskParams struct {
	ID string `json:"id"`
}

// CancelTaskParams is the params object for tasks/cancel.
type CancelTaskParams struct {
	ID string `json:"id"`
}

// ListTasksResult is the result object for tasks/list.
type ListTasksResult struct {
	NextPageToken string `json:"nextPageToken"`
	PageSize      int    `json:"pageSize"`
	Tasks         []Task `json:"tasks"`
	TotalSize     int    `json:"totalSize"`
}
