// Package ollama implements an Ollama LLM provider.
//
// Ollama is a tool for running large language models locally. It provides
// a REST API for generating completions using models like Llama 2, Mistral,
// and many others.
//
// # Basic Usage
//
//	provider, err := ollama.NewProvider(ollama.Config{
//	    BaseURL: "http://localhost:11434", // Optional, this is the default
//	    Model:   "llama2",
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
// # Message Format Conversion
//
// Ollama uses a text-based prompt format instead of structured messages.
// This provider automatically converts the structured message format to
// Ollama's expected format:
//
//	System: You are a helpful assistant
//
//	User: What is the weather?
//
//	Assistant: I don't have access to weather data.
//
//	User: Can you help me?
//
//	Assistant:
//
// # Limitations
//
// - No native function calling support (FunctionCalling capability is false)
// - No vision support
// - Tool calls are converted to text format (best effort)
//
// # Local Deployment
//
// Ollama is designed for local use and doesn't require API keys:
//
//	# Install Ollama
//	curl https://ollama.ai/install.sh | sh
//
//	# Pull a model
//	ollama pull llama2
//
//	# The API is now available at http://localhost:11434
package ollama
