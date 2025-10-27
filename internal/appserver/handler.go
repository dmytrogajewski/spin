package appserver

import (
	"context"
	"encoding/json"

	"github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

// Handler implements jsonrpc.Handler interface
type Handler struct {
	processor *Processor
}

// HandleRequest routes requests to appropriate handlers
func (h *Handler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "initialize":
		var p jsonrpc.InitializeParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		result, err := h.processor.HandleInitialize(ctx, p)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)

	case "send_message":
		var p jsonrpc.SendMessageParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		result, err := h.processor.HandleSendMessage(ctx, p)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)

	case "approve_tool":
		var p jsonrpc.ApproveToolParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		result, err := h.processor.HandleApproveTool(ctx, p)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)

	case "cancel_turn":
		var p jsonrpc.CancelTurnParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		result, err := h.processor.HandleCancelTurn(ctx, p)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)

	case "search_files":
		var p jsonrpc.SearchFilesParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		result, err := h.processor.HandleSearchFiles(ctx, p)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)

	default:
		return nil, jsonrpc.NewError(jsonrpc.MethodNotFound, "method not found")
	}
}
