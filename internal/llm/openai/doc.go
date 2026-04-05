// Package openai implements an OpenAI-compatible LLM provider using the official openai-go SDK.
//
// This provider uses the official OpenAI Go SDK (github.com/openai/openai-go) to
// communicate with OpenAI-compatible APIs. It supports:
//   - OpenAI (https://api.openai.com)
//   - Azure OpenAI
//   - Local providers (Ollama, LMStudio) when configured with compatible endpoints
//   - Any other service implementing the OpenAI Chat Completions API
//
// # Basic Usage
//
//	provider, err := openai.NewProvider(openai.Config{
//	    BaseURL: "https://api.openai.com/v1",
//	    APIKey:  "your-api-key",
//	    Model:   "gpt-4",
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
// # Tool/Function Calling
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
//	resp.ToolCalls contains the function calls to execute.
package openai
