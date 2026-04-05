// Package lmstudio implements an LMStudio LLM provider.
//
// LMStudio is a desktop application for running large language models locally.
// It provides an OpenAI-compatible REST API, making it easy to use existing
// OpenAI-based code with local models.
//
// This provider is a thin wrapper around the OpenAI provider with LMStudio-specific
// defaults (endpoint, no API key).
//
// # Basic Usage
//
//	provider, err := lmstudio.NewProvider(lmstudio.Config{
//
// BaseURL defaults to http://localhost:1234/v1
//
//	    Model: "llama-2-7b-chat",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	req := llm.CompletionRequest{
//	    Messages: []llm.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	}
//
//	resp, err := provider.Complete(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Streaming
//
//	chunks, err := provider.Stream(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for chunk := range chunks {
//	    if chunk.Type == llm.ChunkTypeError {
//	        log.Printf("Error: %v", chunk.Error)
//	        continue
//	    }
//	    fmt.Print(chunk.Content)
//	}
//
// # Function Calling
//
// LMStudio supports function calling through the OpenAI-compatible API:
//
//	req := llm.CompletionRequest{
//	    Messages: []llm.Message{
//	        {Role: "user", Content: "What's the weather in London?"},
//	    },
//	    Tools: []llm.Tool{
//	        {
//	            Type: "function",
//	            Function: llm.Function{
//	                Name:        "get_weather",
//	                Description: "Get current weather",
//	            },
//	        },
//	    },
//	}
//
//	resp, err := provider.Complete(context.Background(), req)
//
// resp.ToolCalls contains the function calls to execute
//
// # Installation & Setup
//
// 1. Download and install LMStudio from https://lmstudio.ai
// 2. Load a model in LMStudio
// 3. Start the local server (default port: 1234)
// 4. The API is now available at http://localhost:1234/v1
//
// # Differences from OpenAI
//
//   - Runs locally (no internet required)
//   - No API key needed
//   - Default endpoint: http://localhost:1234/v1
//   - Limited to models loaded in LMStudio
//   - Typically no vision support
//
// # Differences from Ollama
//
//   - Uses OpenAI API format (structured messages)
//   - Supports function calling
//   - Uses Server-Sent Events for streaming
//   - Different endpoint structure
package lmstudio
