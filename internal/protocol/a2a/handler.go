package a2a

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	taskIDPrefix     = "task-"
	artifactIDEcho   = "artifact-1"
	artifactNameEcho = "result"
	defaultPageSize  = 50
	mediaTextPlain   = "text/plain"
)

// MemoryHandler is an in-memory A2A method implementation for tests and local use.
type MemoryHandler struct {
	card  AgentCard
	mu    sync.Mutex
	seq   atomic.Int64
	tasks map[string]*Task
}

// NewMemoryHandler builds a handler that stores tasks in process memory.
func NewMemoryHandler(card AgentCard) *MemoryHandler {
	return &MemoryHandler{
		card:  card,
		tasks: make(map[string]*Task),
	}
}

// Card returns the handler's Agent Card.
func (handler *MemoryHandler) Card() AgentCard {
	return handler.card
}

// Handle dispatches one A2A method.
func (handler *MemoryHandler) Handle(
	ctx context.Context,
	method string,
	params json.RawMessage,
) (json.RawMessage, *RPCError) {
	if err := ctx.Err(); err != nil {
		return nil, NewRPCError(CodeInternalError, msgInternalError)
	}

	switch method {
	case MethodMessageSend:
		return handler.messageSend(params)
	case MethodMessageStream:
		return handler.messageStream(params)
	case MethodTasksGet:
		return handler.tasksGet(params)
	case MethodTasksList:
		return handler.tasksList()
	case MethodTasksCancel:
		return handler.tasksCancel(params)
	case MethodAgentGetCard:
		return handler.agentGetCard()
	default:
		return nil, NewRPCError(CodeMethodNotFound, msgMethodNotFound)
	}
}

func (handler *MemoryHandler) messageSend(params json.RawMessage) (json.RawMessage, *RPCError) {
	task, rpcErr := handler.createTask(params)
	if rpcErr != nil {
		return nil, rpcErr
	}

	return marshalResult(SendMessageResult{Task: task})
}

func (handler *MemoryHandler) messageStream(params json.RawMessage) (json.RawMessage, *RPCError) {
	if !handler.card.Capabilities.Streaming {
		return nil, NewRPCError(CodeUnsupportedOperation, msgUnsupportedOperation)
	}

	return handler.messageSend(params)
}

func (handler *MemoryHandler) tasksGet(params json.RawMessage) (json.RawMessage, *RPCError) {
	var in GetTaskParams
	if err := json.Unmarshal(params, &in); err != nil || in.ID == "" {
		return nil, NewRPCError(CodeInvalidParams, msgInvalidParams)
	}

	handler.mu.Lock()
	task, ok := handler.tasks[in.ID]
	handler.mu.Unlock()

	if !ok {
		return nil, NewRPCError(CodeTaskNotFound, msgTaskNotFound)
	}

	return marshalResult(*task)
}

func (handler *MemoryHandler) tasksList() (json.RawMessage, *RPCError) {
	handler.mu.Lock()
	listed := make([]Task, 0, len(handler.tasks))

	for _, task := range handler.tasks {
		copyTask := *task
		copyTask.Artifacts = nil
		listed = append(listed, copyTask)
	}

	slices.SortFunc(listed, func(left, right Task) int {
		return cmp.Compare(left.ID, right.ID)
	})

	handler.mu.Unlock()

	return marshalResult(ListTasksResult{
		Tasks:         listed,
		NextPageToken: "",
		PageSize:      defaultPageSize,
		TotalSize:     len(listed),
	})
}

func (handler *MemoryHandler) tasksCancel(params json.RawMessage) (json.RawMessage, *RPCError) {
	var in CancelTaskParams
	if err := json.Unmarshal(params, &in); err != nil || in.ID == "" {
		return nil, NewRPCError(CodeInvalidParams, msgInvalidParams)
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	task, ok := handler.tasks[in.ID]
	if !ok {
		return nil, NewRPCError(CodeTaskNotFound, msgTaskNotFound)
	}

	if task.Status.State.Terminal() {
		return nil, NewRPCError(CodeTaskNotCancelable, msgTaskNotCancelable)
	}

	task.Status.State = TaskStateCanceled

	return marshalResult(*task)
}

func (handler *MemoryHandler) agentGetCard() (json.RawMessage, *RPCError) {
	if !handler.card.Capabilities.ExtendedAgentCard {
		return nil, NewRPCError(CodeUnsupportedOperation, msgUnsupportedOperation)
	}

	return marshalResult(handler.card)
}

func (handler *MemoryHandler) createTask(params json.RawMessage) (*Task, *RPCError) {
	var in SendMessageParams
	if err := json.Unmarshal(params, &in); err != nil || in.Message.MessageID == "" || len(in.Message.Parts) == 0 {
		return nil, NewRPCError(CodeInvalidParams, msgInvalidParams)
	}

	state := TaskStateCompleted
	if in.Configuration != nil && in.Configuration.ReturnImmediately {
		state = TaskStateWorking
	}

	task := &Task{
		ID:      handler.nextTaskID(),
		Status:  TaskStatus{State: state},
		History: []Message{in.Message},
		Artifacts: []Artifact{{
			ArtifactID: artifactIDEcho,
			Name:       artifactNameEcho,
			Parts:      []Part{{Text: firstText(in.Message), MediaType: mediaTextPlain}},
		}},
	}

	handler.mu.Lock()
	handler.tasks[task.ID] = task
	handler.mu.Unlock()

	return task, nil
}

func (handler *MemoryHandler) nextTaskID() string {
	return taskIDPrefix + strconv.FormatInt(handler.seq.Add(1), 10)
}

func firstText(message Message) string {
	for _, part := range message.Parts {
		if part.Text != "" {
			return part.Text
		}
	}

	return ""
}

func marshalResult(value any) (json.RawMessage, *RPCError) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, NewRPCError(CodeInternalError, msgInternalError)
	}

	return body, nil
}
