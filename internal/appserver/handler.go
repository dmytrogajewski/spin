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
func (h *Handler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "initialize":
		var p jsonrpc.InitializeParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		return h.processor.HandleInitialize(ctx, p)

	case "send_message":
		var p jsonrpc.SendMessageParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		return h.processor.HandleSendMessage(ctx, p)

	case "approve_tool":
		var p jsonrpc.ApproveToolParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		return h.processor.HandleApproveTool(ctx, p)

	case "cancel_turn":
		var p jsonrpc.CancelTurnParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		return h.processor.HandleCancelTurn(ctx, p)

	case "search_files":
		var p jsonrpc.SearchFilesParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, jsonrpc.NewError(jsonrpc.InvalidParams, "invalid parameters")
		}
		return h.processor.HandleSearchFiles(ctx, p)

	default:
		return nil, jsonrpc.NewError(jsonrpc.MethodNotFound, "method not found")
	}
}
