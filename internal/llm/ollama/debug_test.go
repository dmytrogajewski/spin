package ollama

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openai/openai-go"
)

func TestOllamaStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider, err := NewProvider(Config{
		BaseURL: "http://localhost:11434",
		Model:   "qwen2.5-coder:1.5b",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Count to 3"),
		}),
	}

	ctx := context.Background()
	chunks, err := provider.Stream(ctx, params)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("\n=== Streaming response ===")
	content := ""
	chunkCount := 0
	for chunk := range chunks {
		chunkCount++

		// Extract content from chunk
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fmt.Print(delta.Content)
				content += delta.Content
			}

			// Check for finish
			if chunk.Choices[0].FinishReason != "" {
				fmt.Printf("\n[Finish reason: %s]\n", chunk.Choices[0].FinishReason)
			}
		}
	}
	fmt.Println("\n=== Done ===")
	fmt.Printf("Total chunks: %d\n", chunkCount)
	fmt.Printf("Total content length: %d\n", len(content))
	fmt.Printf("Content: %q\n", content)

	if content == "" {
		t.Fatal("No content received from stream")
	}
}
