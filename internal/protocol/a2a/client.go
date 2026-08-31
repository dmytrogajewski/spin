package a2a

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// Client is a sequential JSON-RPC 2.0 client for the local A2A binding.
type Client struct {
	card   AgentCard
	mu     sync.Mutex
	nextID atomic.Int64
	reader *bufio.Reader
	writer io.Writer
}

// NewClient reads the first framed Agent Card announce, then is ready to Call.
func NewClient(reader io.Reader, writer io.Writer) (*Client, error) {
	client := &Client{
		reader: bufio.NewReader(reader),
		writer: writer,
	}

	env, err := readEnvelope(client.reader)
	if err != nil {
		return nil, fmt.Errorf("read card: %w", err)
	}

	if env.Method != MethodAgentCard {
		return nil, fmt.Errorf("%w: %s", ErrUnexpectedCard, env.Method)
	}

	if unmarshalErr := json.Unmarshal(env.Params, &client.card); unmarshalErr != nil {
		return nil, fmt.Errorf("decode card: %w", unmarshalErr)
	}

	return client, nil
}

// Card returns the Agent Card announced on connect.
func (client *Client) Card() AgentCard {
	return client.card
}

// Call sends a JSON-RPC request and decodes the result into result.
func (client *Client) Call(ctx context.Context, method string, params, result any) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("call %s: %w", method, err)
	}

	paramsRaw, marshalErr := json.Marshal(params)
	if marshalErr != nil {
		return fmt.Errorf("marshal %s params: %w", method, marshalErr)
	}

	idRaw, idErr := json.Marshal(client.nextID.Add(1))
	if idErr != nil {
		return fmt.Errorf("marshal %s id: %w", method, idErr)
	}

	if writeErr := writeEnvelope(client.writer, envelope{
		JSONRPC: jsonRPCVersion,
		ID:      idRaw,
		Method:  method,
		Params:  paramsRaw,
	}); writeErr != nil {
		return fmt.Errorf("write %s: %w", method, writeErr)
	}

	return client.readCallResult(method, result)
}

func (client *Client) readCallResult(method string, result any) error {
	env, readErr := readEnvelope(client.reader)
	if readErr != nil {
		return fmt.Errorf("read %s: %w", method, readErr)
	}

	if env.Error != nil {
		return env.Error
	}

	if result == nil || len(env.Result) == 0 {
		return nil
	}

	if unmarshalErr := json.Unmarshal(env.Result, result); unmarshalErr != nil {
		return fmt.Errorf("decode %s result: %w", method, unmarshalErr)
	}

	return nil
}

// SendMessage calls message/send and returns the created Task.
func (client *Client) SendMessage(ctx context.Context, message Message) (*Task, error) {
	return client.send(ctx, MethodMessageSend, SendMessageParams{Message: message})
}

// SendMessageImmediate calls message/send with returnImmediately so the task stays working.
func (client *Client) SendMessageImmediate(ctx context.Context, message Message) (*Task, error) {
	return client.send(ctx, MethodMessageSend, SendMessageParams{
		Message:       message,
		Configuration: &SendMessageConfiguration{ReturnImmediately: true},
	})
}

// StreamMessage calls message/stream.
func (client *Client) StreamMessage(ctx context.Context, message Message) (*Task, error) {
	return client.send(ctx, MethodMessageStream, SendMessageParams{Message: message})
}

func (client *Client) send(ctx context.Context, method string, params SendMessageParams) (*Task, error) {
	var out SendMessageResult
	if err := client.Call(ctx, method, params, &out); err != nil {
		return nil, err
	}

	if out.Task == nil {
		return nil, NewRPCError(CodeInvalidAgentResponse, msgInvalidAgentResponse)
	}

	return out.Task, nil
}

// GetTask calls tasks/get.
func (client *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	var task Task
	if err := client.Call(ctx, MethodTasksGet, GetTaskParams{ID: taskID}, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// ListTasks calls tasks/list.
func (client *Client) ListTasks(ctx context.Context) (*ListTasksResult, error) {
	var out ListTasksResult
	if err := client.Call(ctx, MethodTasksList, struct{}{}, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// CancelTask calls tasks/cancel.
func (client *Client) CancelTask(ctx context.Context, taskID string) (*Task, error) {
	var task Task
	if err := client.Call(ctx, MethodTasksCancel, CancelTaskParams{ID: taskID}, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// GetCard calls agent/getAuthenticatedExtendedCard.
func (client *Client) GetCard(ctx context.Context) (*AgentCard, error) {
	var card AgentCard
	if err := client.Call(ctx, MethodAgentGetCard, struct{}{}, &card); err != nil {
		return nil, err
	}

	return &card, nil
}
