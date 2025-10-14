# Cycle Auto-Discovery

**Feature:** Automatic detection and intervention for agent reasoning loops

**Status:** ✅ Implemented (Feature 4 of Advanced Features Roadmap)

**Package:** `internal/core/cycle/`

---

## Overview

Cycle Auto-Discovery is an intelligent system that automatically detects when Spin agents get stuck in infinite reasoning loops and applies appropriate interventions to restore productive behavior.

### What It Detects

The system identifies several types of problematic patterns:

- **Similar Responses**: When the LLM repeats similar responses (>80% similarity)
- **Repeated Tools**: When the same tool is called repeatedly without progress
- **State Oscillation**: When the agent alternates between two states (A→B→A→B)
- **Error Loops**: When identical errors occur multiple times

### Intervention Strategies

When cycles are detected, the system applies interventions based on severity:

- **Reflection (Soft)**: Injects prompts encouraging alternative approaches
- **Context Summarization (Medium)**: Compresses conversation history to refocus
- **User Escalation (Hard)**: Pauses execution and requests user guidance

## Configuration

### YAML Configuration

Add cycle detection settings to your `~/.spin/config.yaml`:

```yaml
agent:
  cycle_detection:
    enabled: true              # Enable/disable cycle detection (default: true)
    window_size: 3             # Number of snapshots to analyze (default: 3)
    similarity_threshold: 0.8  # Similarity threshold for response detection (default: 0.8)
    tool_repeat_limit: 3       # Max identical tool calls before cycle (default: 3)
    error_repeat_limit: 3      # Max identical errors before cycle (default: 3)
```

### Programmatic Configuration

```go
// Create agent with custom cycle detection settings
agent, err := core.NewAgent(
    llmProvider,
    executor,
    validator,
    environment,
    eventEmitter,
    core.WithAgentConfig(&core.AgentConfig{
        CycleDetection: struct {
            Enabled         bool
            WindowSize      int
            SimilarityThresh float64
            ToolRepeatLimit  int
            ErrorRepeatLimit int
        }{
            Enabled:         true,
            WindowSize:      5,
            SimilarityThresh: 0.85,
            ToolRepeatLimit:  4,
            ErrorRepeatLimit: 2,
        },
    }),
)
```

## How It Works

### 1. Snapshot Recording

After each agent turn, the system records a snapshot of the current state:

```go
snapshot := cycle.Snapshot{
    Turn:      turnNumber,
    Response:  llmResponse.Content,
    ToolCalls: extractToolNames(llmResponse.ToolCalls),
    Error:     errorMessage,
    Timestamp: time.Now(),
}
detector.Record(snapshot)
```

### 2. Pattern Detection

The detector analyzes recent snapshots for problematic patterns:

- **Similarity Check**: Compares consecutive responses using Jaccard similarity
- **Tool Pattern**: Identifies repeated tool usage
- **Oscillation**: Detects alternating state patterns
- **Error Pattern**: Catches repeated identical errors

### 3. Intervention Application

When a cycle is detected, the appropriate intervention is applied:

- **Early cycles** (<10 turns): Reflection prompts
- **Mid-stage cycles** (10-30 turns): Context summarization
- **Late cycles** (>30 turns): User escalation

### 4. Event Emission

Cycle detection events are emitted for status bar integration:

```go
// Example event emission
emitter.Emit(core.Event{
    Type: core.EventWarning,
    Data: core.SystemEventData{
        Level:   "warning",
        Message: "Cycle detected: similar_responses. Applied intervention: Reflection",
        Details: "Detected 3 similar consecutive responses",
    },
})
```

## Usage Examples

### Basic Usage

Cycle detection works automatically when enabled. No additional code is required - the agent loop handles detection and intervention transparently.

### Custom Detector

For advanced use cases, you can create and use a custom detector:

```go
// Create detector with custom configuration
config := cycle.Config{
    WindowSize:       5,
    SimilarityThresh: 0.9,
    ToolRepeatLimit:  2,
    Enabled:         true,
}

detector := cycle.NewDetector(config)

// Use in custom agent implementation
detector.Record(snapshot)
result, err := detector.Check()
if result.Type != cycle.CycleNone {
    // Handle cycle detection
}
```

### Manual Intervention

You can also apply interventions manually:

```go
// Create intervention selector
selector := cycle.NewInterventionSelector()

// Select appropriate intervention
intervention := selector.SelectIntervention(cycle.CycleRepeatedTool, turnCount)

// Apply intervention
modifiedMessages, err := intervention.Apply(ctx, conversationMessages)
```

## Performance

### Benchmarks

- **Detection latency**: <10ms per check
- **Memory overhead**: <1MB for typical usage (configurable window size)
- **No blocking operations** in detection logic

### Tuning

For high-frequency scenarios, adjust these settings:

```yaml
agent:
  cycle_detection:
    window_size: 2        # Smaller window for faster detection
    tool_repeat_limit: 5   # Allow more tool repetitions before cycle
```

For high-precision scenarios:

```yaml
agent:
  cycle_detection:
    similarity_threshold: 0.9  # Higher threshold for stricter detection
    error_repeat_limit: 2      # Stricter error detection
```

## Troubleshooting

### Common Issues

#### False Positives

**Problem**: Legitimate similar responses flagged as cycles

**Solution**: Increase similarity threshold or window size:

```yaml
agent:
  cycle_detection:
    similarity_threshold: 0.9
    window_size: 5
```

#### Missing Detection

**Problem**: Actual cycles not being detected

**Solution**: Decrease thresholds for more sensitive detection:

```yaml
agent:
  cycle_detection:
    similarity_threshold: 0.7
    tool_repeat_limit: 2
```

#### Performance Impact

**Problem**: Cycle detection slowing down agent execution

**Solution**: Disable detection or reduce window size:

```yaml
agent:
  cycle_detection:
    enabled: false  # Disable entirely
    # OR
    window_size: 2  # Reduce analysis window
```

### Debugging

Enable debug logging to see cycle detection activity:

```bash
# Run with debug logging
SPIN_LOG_LEVEL=debug spin [command]

# Look for cycle detection messages:
# [DEBUG] cycle detector: recorded snapshot turn=5
# [WARN] cycle detected: similar_responses confidence=0.85
# [INFO] applied intervention: Reflection
```

### Manual Testing

Test cycle detection with synthetic scenarios:

```bash
# Create a script that generates repeated patterns
echo "read_file
read_file
read_file" | spin exec "repeat this pattern"
```

## Integration

### Status Bar Integration

Cycle detection events are automatically displayed in the status bar:

```
[●] 42% Planning ollama/qwen3:1.7b 125 tok/s conv:abc123 ⚠ Cycle: similar_responses
```

### Event Types

The system emits these event types:

- `EventWarning`: When cycles are detected and interventions applied
- `EventTurnPaused`: When user escalation occurs
- `EventInfo`: When interventions are successfully applied

### Metrics

Monitor cycle detection effectiveness:

```go
// Access detector history for metrics
history := detector.GetHistory()

// Calculate detection rate
cyclesDetected := 0
for _, snapshot := range history {
    // Analyze for cycle patterns
}
```

## Best Practices

### Configuration

- **Start conservative**: Use default settings initially
- **Tune for your use case**: Adjust thresholds based on your specific scenarios
- **Monitor performance**: Watch for detection latency in high-frequency scenarios

### Error Handling

- **Graceful degradation**: Agent continues functioning even if cycle detection fails
- **User override**: Users can always intervene manually via approval dialogs
- **Logging**: All cycle detection activity is logged for debugging

### Testing

- **Unit tests**: Test individual detection methods with synthetic data
- **Integration tests**: Test end-to-end cycle detection in realistic scenarios
- **Performance tests**: Verify detection latency meets requirements

## Examples

### Example 1: Repeated Tool Detection

```bash
# Agent gets stuck in a loop calling the same tool
$ spin exec "list the directory, then list it again, then list it once more"

# Cycle detection identifies the pattern:
# [WARN] Cycle detected: repeated_tool 'list_dir' called 3 times
# [INFO] Applied intervention: Reflection

# Agent receives reflection prompt and continues productively
```

### Example 2: Similar Response Detection

```bash
# Agent repeats similar responses
$ spin exec "analyze this code and then analyze it again"

# Cycle detection identifies similarity:
# [WARN] Cycle detected: similar_responses confidence=0.87
# [INFO] Applied intervention: Reflection

# Agent gets reflection prompt and tries different approach
```

### Example 3: User Escalation

```bash
# Agent stuck in persistent cycle
$ spin exec "keep trying the same failing approach"

# After multiple failed interventions:
# [WARN] Cycle detected: oscillation pattern
# [ERROR] Applied intervention: User Escalation
# [INFO] Agent paused - awaiting user guidance

# User sees approval dialog and can provide guidance
```

## API Reference

### Core Types

```go
// Cycle detection result
type CycleResult struct {
    Type       CycleType    // Type of cycle detected
    Confidence float64      // Detection confidence (0.0-1.0)
    Details    string       // Human-readable description
    Timestamp  time.Time    // When cycle was detected
}

// Cycle types
type CycleType int
const (
    CycleNone           CycleType = iota
    CycleSimilarResponses
    CycleRepeatedTool
    CycleOscillation
    CycleSameError
)

// Intervention result
type InterventionResult struct {
    Type      InterventionType
    Success   bool
    Message   string
    Timestamp time.Time
}
```

### Detector Interface

```go
type Detector interface {
    Record(snapshot Snapshot)                // Record agent state
    Check() (CycleResult, error)             // Check for cycles
    GetHistory() []Snapshot                  // Get detection history
    Reset()                                  // Clear history
}
```

### Intervention Interface

```go
type Intervention interface {
    Apply(ctx context.Context, messages []Message) ([]Message, error)
    Name() string
    Description() string
    Severity() int  // 1=Soft, 2=Medium, 3=Hard
}
```

## Future Enhancements

### Planned Features

- **Sliding Window Compression**: Keep last N messages verbatim, compress older
- **Semantic Similarity**: Use embeddings for better similarity detection
- **Pattern Learning**: Learn from successful interventions
- **Integration with Context Summarization**: Use existing compression for medium interventions

### Extension Points

The system is designed for extensibility:

- **Custom Detectors**: Implement the `Detector` interface for specialized detection
- **Custom Interventions**: Implement the `Intervention` interface for new strategies
- **Event Integration**: Hook into the event system for custom monitoring

---

**Related Documentation:**
- [Advanced Features Roadmap](../specs/advanced-features-20251012/ROADMAP.md)
- [Core Package Documentation](packages/core.md)
- [Agent Configuration](packages/core.md#configuration)

**Implementation:**
- [FRD Document](../specs/frds/FRD-20251015020000-cycle-auto-discovery.md)
- [Package Source](../../../internal/core/cycle/)

*Last Updated: 2025-10-15*  
*Feature Status: ✅ Complete*
