// Package main demonstrates custom tool registration with the Spin core package.
//
// This example shows how to:
//   - Define a custom tool
//   - Register it with the tool registry
//   - Use it in a conversation
//
// Run this example:
//
//	go run examples/custom-tool/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func main() {
	fmt.Println("🔧 Spin AI Agent - Custom Tool Example")
	fmt.Println("=" + string(make([]byte, 48)))
	fmt.Println()

	// Create tool registry
	registry := tools.NewRegistry()

	// Register custom tools
	registerCustomTools(registry)

	// List registered tools
	fmt.Println("Registered tools:")
	for _, toolName := range []string{"calculate", "get_weather", "get_time"} {
		tool, exists := registry.Get(toolName)
		if exists {
			fmt.Printf("  ✓ %s: %s\n", tool.Name, tool.Description)
		}
	}
	fmt.Println()

	// Create configuration
	cfg := &core.Config{
		MaxTurns:    5,
		Temperature: 0.7,
		MaxTokens:   1024,
		Model:       "claude-3-5-sonnet-20241022",
		Debug:       true,
	}

	// Create manager
	mgr, err := core.NewManager(
		cfg,
		core.WithLLMProvider(&MockLLMProvider{}),
		core.WithToolRegistry(registry),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Start conversation
	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx)
	if err != nil {
		log.Fatalf("Failed to create conversation: %v", err)
	}

	// Demonstrate tool usage
	demonstrateCalculator(ctx, conv)
	demonstrateWeather(ctx, conv)
	demonstrateTime(ctx, conv)

	fmt.Println("\nExample completed successfully!")
}

func registerCustomTools(registry *tools.Registry) {
	// Calculator tool
	calcTool := &tools.Tool{
		Name:        "calculate",
		Description: "Performs basic arithmetic calculations (add, subtract, multiply, divide)",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"operation": {
					Type:        "string",
					Description: "The operation to perform: add, subtract, multiply, divide",
					Enum:        []string{"add", "subtract", "multiply", "divide"},
				},
				"a": {
					Type:        "number",
					Description: "First number",
				},
				"b": {
					Type:        "number",
					Description: "Second number",
				},
			},
			Required: []string{"operation", "a", "b"},
		},
	}
	registry.Register(calcTool)

	// Weather tool
	weatherTool := &tools.Tool{
		Name:        "get_weather",
		Description: "Gets the current weather for a location",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"location": {
					Type:        "string",
					Description: "City name or location",
				},
			},
			Required: []string{"location"},
		},
	}
	registry.Register(weatherTool)

	// Time tool
	timeTool := &tools.Tool{
		Name:        "get_time",
		Description: "Gets the current time in a specific timezone",
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"timezone": {
					Type:        "string",
					Description: "Timezone name (e.g., 'America/New_York', 'UTC')",
				},
			},
			Required: []string{"timezone"},
		},
	}
	registry.Register(timeTool)
}

func demonstrateCalculator(ctx context.Context, conv *core.Conversation) {
	fmt.Println("📊 Calculator Tool Demo")
	fmt.Println("-" + string(make([]byte, 24)))

	// Simulate a calculation
	args := map[string]interface{}{
		"operation": "multiply",
		"a":         float64(42),
		"b":         float64(7),
	}

	result := executeCalculator(args)
	fmt.Printf("Calculate: 42 × 7 = %v\n\n", result)
}

func demonstrateWeather(ctx context.Context, conv *core.Conversation) {
	fmt.Println("🌤️  Weather Tool Demo")
	fmt.Println("-" + string(make([]byte, 24)))

	args := map[string]interface{}{
		"location": "San Francisco",
	}

	result := executeWeather(args)
	fmt.Printf("Weather: %v\n\n", result)
}

func demonstrateTime(ctx context.Context, conv *core.Conversation) {
	fmt.Println("🕐 Time Tool Demo")
	fmt.Println("-" + string(make([]byte, 24)))

	args := map[string]interface{}{
		"timezone": "America/New_York",
	}

	result := executeTime(args)
	fmt.Printf("Time: %v\n\n", result)
}

// Tool execution functions (these would be called by the core executor)

func executeCalculator(args map[string]interface{}) interface{} {
	operation := args["operation"].(string)
	a := args["a"].(float64)
	b := args["b"].(float64)

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return map[string]string{"error": "division by zero"}
		}
		result = a / b
	default:
		return map[string]string{"error": "unknown operation"}
	}

	return map[string]float64{"result": result}
}

func executeWeather(args map[string]interface{}) interface{} {
	location := args["location"].(string)

	// Mock weather data
	return map[string]interface{}{
		"location":    location,
		"temperature": 72,
		"condition":   "sunny",
		"humidity":    45,
		"wind_speed":  8,
		"units":       "imperial",
	}
}

func executeTime(args map[string]interface{}) interface{} {
	timezone := args["timezone"].(string)

	// Mock time data
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return map[string]string{"error": "invalid timezone"}
	}

	currentTime := time.Now().In(loc)
	return map[string]string{
		"timezone": timezone,
		"time":     currentTime.Format("15:04:05"),
		"date":     currentTime.Format("2006-01-02"),
		"offset":   currentTime.Format("-07:00"),
	}
}

// MockLLMProvider for demonstration
type MockLLMProvider struct{}

func (m *MockLLMProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Content:      "Tool execution completed",
		StopReason:   llm.StopReasonEndTurn,
		TokensInput:  30,
		TokensOutput: 20,
	}, nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamEvent, error) {
	eventChan := make(chan llm.StreamEvent, 1)
	go func() {
		defer close(eventChan)
		eventChan <- llm.StreamEvent{
			Type:       llm.StreamEventTypeDone,
			StopReason: llm.StopReasonEndTurn,
		}
	}()
	return eventChan, nil
}

func prettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
