# Research: Advanced TUI and Agent Features for Spin

**Date:** 2025-10-12
**Researcher:** AI Research Agent
**Status:** Research Phase Complete
**Methodology:** TRIZ-based systematic innovation

---

## Executive Summary

This document presents comprehensive research and TRIZ-based solution design for five advanced features requested for the Spin coding agent:

1. **Persistent Status Bar** - Real-time metrics display (context %, agent state, provider, tokens/sec, etc.)
2. **Context Summarization** - Automatic context compression to prevent overflow
3. **VRAM Auto-Tuning** - Intelligent model parameter adjustment based on available VRAM
4. **Cycle Auto-Discovery** - Detection of agent reasoning loops and automatic intervention
5. **Enhanced Approval Mechanisms** - User control over command execution

Each feature has been analyzed using TRIZ methodology, with web research, competitor analysis, and deep codebase understanding informing the solutions.

---

## Table of Contents

1. [Feature 1: Persistent Status Bar](#feature-1-persistent-status-bar)
2. [Feature 2: Context Summarization](#feature-2-context-summarization)
3. [Feature 3: VRAM Auto-Tuning](#feature-3-vram-auto-tuning)
4. [Feature 4: Cycle Auto-Discovery](#feature-4-cycle-auto-discovery)
5. [Feature 5: Enhanced Approval Mechanisms](#feature-5-enhanced-approval-mechanisms)
6. [Implementation Roadmap](#implementation-roadmap)
7. [References](#references)

---

## Feature 1: Persistent Status Bar

### Problem Statement

Users need real-time visibility into agent operation without disrupting the native scrollback TUI. The status bar must display:
- Context fill percentage (current/max tokens)
- Agent activity state ("Calling tools", "Summarizing", "Planning", "Thinking", "Loading", "Waiting approval")
- Current task mode (regular, review, compact, planning)
- Hotkey information
- Current provider + model
- Tokens per second throughput
- Conversation ID

### Research Findings

#### Industry Solutions (2024)

**Modern TUI Frameworks:**
- **Bubble Tea** (Go): MUV architecture, real-time system monitoring
- **Ratatui**: Feature-rich with tables, charts, scrollbars
- **s-tui**: CPU monitoring with sensors displayed at bottom in text form
- **gonzo**: Real-time log analysis with status updates

**Key Patterns:**
- Bottom-anchored status bars that persist across scrolling
- Non-blocking async updates
- Compact single-line or dual-line layouts
- ANSI escape sequence positioning

#### Competitor Analysis

**Claude Code:**
- Minimal status feedback (mainly streaming text)
- No persistent metrics display
- Opportunity: Spin can exceed with visibility

**Cursor:**
- Real-time code completion metrics
- Resource usage hints
- Checkpoint status display

**Aider:**
- Git-integrated status (branch, changes)
- Model selection visible in prompt

### Current Codebase Analysis

**Existing TUI Architecture:**
```
internal/ui/
├── term/       # Terminal control (raw mode, ANSI)
├── prompt/     # Prompt subsystem
├── output/     # Append-only output
├── blocks/     # Block timeline
└── adapters/   # PureTTY adapter
```

**Relevant Components:**
- `internal/ui/term/tty.go`: ANSI control, cursor positioning
- `internal/ui/prompt/renderer.go`: Prompt redraw logic
- `internal/ui/output/coordinator.go`: Output + prompt coordination
- `internal/core/event.go`: Event system with real-time data

**Token Tracking:**
- `internal/core/history.go`: Has `TokensUsed` in history
- `internal/core/tokenizer.go`: Token counting logic
- `internal/llm/tokenizer.go`: Model-specific tokenizers

**Event System:**
- `TurnEventData`: Contains `TokensUsed`, `TurnsUsed`, `MaxTurns`
- Real-time events already flow through `EventEmitter`

### TRIZ Analysis

#### System Function
**Input:** Agent events, metrics data → **Function:** Display status → **Output:** Real-time user visibility

#### Main Contradiction
**Improve** real-time status visibility **without worsening** native scrollback preservation (Factory Droid principle)

#### Ideal Final Result (IFR)
Status bar updates automatically from event stream, never disrupts scrollback, uses zero full-screen repainting, adapts to terminal width.

#### TRIZ Principles Applied

1. **Segmentation** → Separate status bar from timeline
   - Status bar is independent rendering component
   - Updates via separate ANSI escape sequences

2. **Local Quality** → Optimize status bar rendering separately
   - Minimal redraws (only on data change)
   - Efficient string formatting

3. **Parameter Change** → Use terminal capabilities detection
   - Adapt layout to terminal width
   - Degrade gracefully (hide less critical fields)

4. **Feedback** → Self-updating from event stream
   - Subscribe to `EventEmitter`
   - No polling, fully event-driven

5. **Dynamization** → Adaptive field display
   - Show/hide fields based on relevance
   - Compact mode when terminal is narrow

6. **Nested Doll** → Hierarchical metric structure
   - Primary metrics (always visible): state, context %
   - Secondary metrics (show if space): provider, tokens/sec
   - Tertiary (overflow menu): conversation ID, full model name

### Design Options

#### Option A: Single-Line Bottom Status Bar
**Pros:**
- Minimal vertical space (1 line)
- Aligns with Factory Droid (append-only)
- Easy ANSI implementation (position cursor above prompt)

**Cons:**
- Limited horizontal space for all metrics
- Requires smart truncation logic

**Reversibility:** High (just stop rendering it)

**Example Layout:**
```
┌────────────────────────────────────────────────────────────────────┐
│ [●] 42%  Planning  ollama/qwen3:1.7b  125 tok/s  conv:abc123  ?:help │
└────────────────────────────────────────────────────────────────────┘
> _
```

#### Option B: Dual-Line Status Bar
**Pros:**
- More space for all metrics
- Can use top line for primary, bottom for secondary

**Cons:**
- Takes 2 lines of vertical space
- Slightly more complex rendering

**Reversibility:** High

**Example Layout:**
```
┌────────────────────────────────────────────────────────────────────┐
│ [●] Calling tools │ Context: 42% (8.5K/20K) │ Mode: regular       │
│ Provider: ollama/qwen3:1.7b │ 125 tok/s │ ID: abc123 │ ?: help    │
└────────────────────────────────────────────────────────────────────┘
> _
```

#### Option C: Right-Aligned Transient Status
**Pros:**
- Doesn't take dedicated line
- Minimal visual clutter
- Already contemplated in TUI docs

**Cons:**
- Less visible
- Can overlap with long prompts
- Harder to show all metrics

**Reversibility:** High

**Example Layout:**
```
> user typing here...                    [●] 42% Planning 125tok/s
```

### Selected Option: A (Single-Line Bottom Status Bar)

**Rationale:**
- Best balance of visibility and space efficiency
- Aligns with s-tui, htop, and other successful TUI tools
- Native to Factory Droid principle (doesn't disrupt scrollback)
- Simple implementation using existing ANSI infrastructure

**Expected Effects:**
- Users gain constant awareness of agent state
- No performance impact (event-driven updates)
- Terminal-agnostic (works in SSH, tmux, screen)

### Implementation Plan

**1. Create StatusBar Component** (`internal/ui/statusbar/`)
```go
type StatusBar struct {
    metrics  Metrics
    renderer *Renderer
    mu       sync.RWMutex
}

type Metrics struct {
    AgentState       string  // "Calling tools", "Thinking", etc.
    ContextUsed      int
    ContextMax       int
    TaskMode         string
    Provider         string
    Model            string
    TokensPerSec     float64
    ConversationID   string
}

func (s *StatusBar) Update(m Metrics)
func (s *StatusBar) Render(w io.Writer, width int) error
```

**2. Integrate with Coordinator** (`internal/ui/output/coordinator.go`)
- Add status bar between output and prompt
- Sequence: `[Output] → [StatusBar] → [Prompt]`
- Use ANSI cursor positioning

**3. Wire to Event System** (`internal/core/tui_mapper.go`)
- Subscribe to events: `EventTurnStart`, `EventToolCallStart`, `EventContentDelta`
- Extract metrics from event data
- Call `StatusBar.Update()`

**4. Add Metrics Calculation**
- Context %: `(history.TokensUsed / taskMode.MaxTokens) * 100`
- Tokens/sec: Track delta between `EventContentDelta` timestamps
- Agent state: Map event types to readable strings

**5. Terminal Width Adaptation**
```go
func (r *Renderer) Render(metrics Metrics, width int) string {
    if width < 60 {
        // Minimal: [●] 42% Planning
        return renderCompact(metrics)
    } else if width < 100 {
        // Medium: [●] 42% Planning ollama/qwen 125tok/s
        return renderMedium(metrics)
    } else {
        // Full: All fields
        return renderFull(metrics)
    }
}
```

**6. Hotkey Support**
- `?` → Toggle hotkey help overlay
- Render shortcuts: `^C:quit ^D:exit ^P:palette`

### Testing Plan

**Unit Tests:**
- `statusbar_test.go`: Metric updates, rendering with various widths
- `statusbar_renderer_test.go`: ANSI escape correctness, truncation

**Integration Tests:**
- `statusbar_integration_test.go`: Wire with event emitter
- Verify metrics accuracy

**E2E Tests:**
- `e2e/statusbar_test.go`: Visual validation in terminal
- Test in tmux, SSH, different terminal sizes

### Learnings

**New Reusable Pattern:**
- Event-driven metrics aggregation for TUI components
- Adaptive layout based on terminal capabilities

---

## Feature 2: Context Summarization

### Problem Statement

LLM context windows have hard limits (e.g., 16K, 128K tokens). Long conversations overflow, causing:
- Context truncation (losing important history)
- Performance degradation
- Potential failures

Need automatic strategies to compress context while preserving semantic meaning and task continuity.

### Research Findings

#### State-of-the-Art Techniques (2024-2025)

**1. Semantic Compression**
- **Method:** Pre-trained model reduces semantic redundancy
- **Results:** 6-8x compression without fine-tuning
- **Source:** "Extending Context Window via Semantic Compression" (arXiv 2312.09571v1)

**2. Prompt Compression (LLMLingua)**
- **Method:** Compress prompts up to 20x while preserving ICL and reasoning
- **LongLLMLingua:** Designed for RAG and long-context scenarios
- **Source:** Microsoft Research 2024

**3. Soft Prompt Compression**
- **Method:** Natural language summarization + trainable soft prompts
- **Benefit:** Distills lengthy texts into compact summaries

**4. In-Context Former (IC-Former)**
- **Method:** Cross-attention blocks with learnable digest tokens
- **Complexity:** Linear growth with context length (O(n))
- **Results:** Extract information into compact digest vectors

**5. Hierarchical Processing**
- **Method:** Group tokens hierarchically (token→sentence→paragraph)
- **Benefit:** Operate on compressed representations

**6. Recurrent Context Compression**
- **Method:** Efficiently expand context window via recurrent compression
- **Source:** arXiv 2406.06110v1

**7. Progressive Summarization**
- **Method:** Iteratively summarize chunks when context exceeds limit
- **Pattern:** Divide → Compress → Merge

### Current Codebase Analysis

**Existing History Management:**
```go
// internal/core/history.go
type History struct {
    messages     []Message
    tokensUsed   int
    maxTokens    int  // Current task mode limit
    tokenizer    Tokenizer
}

func (h *History) Truncate(maxTokens int)  // Already exists!
```

**Token Tracking:**
- `internal/core/tokenizer.go`: Model-agnostic token counting
- `internal/llm/tokenizer.go`: Provider-specific tokenizers
- Token counts stored in `TurnEventData.TokensUsed`

**Task Modes:**
- Regular: 16K tokens
- Review: 12K tokens
- Compact: 4K tokens
- Planning: 4K tokens

### TRIZ Analysis

#### System Function
**Input:** Growing message history → **Function:** Compress context → **Output:** Condensed history within token limit

#### Main Contradiction
**Improve** context retention **without worsening** token limit constraints

#### Ideal Final Result (IFR)
History automatically compresses when approaching limits, preserving critical information, with zero manual intervention.

#### TRIZ Principles Applied

1. **Prior Action** → Preemptive compression
   - Trigger compression at 80% capacity (not 100%)
   - Avoid emergency truncation

2. **Segmentation** → Multi-level summarization
   - Summary level 1: Per-turn summaries
   - Summary level 2: Multi-turn summaries
   - Summary level 3: Session summary

3. **Nested Doll** → Hierarchical compression
   - Keep recent messages verbatim
   - Summarize mid-range messages
   - Ultra-compress old messages

4. **Universality** → Pluggable strategies
   - Interface-based: multiple compression strategies
   - Strategy registry pattern

5. **Parameter Change** → Adaptive compression ratio
   - Aggressive compression when critically full
   - Light compression when moderately full

6. **Feedback** → Self-monitoring system
   - Track compression effectiveness (semantic loss metric)
   - Auto-adjust strategy if quality degrades

### Design Options

#### Option A: Simple Sliding Window
**Summary:** Keep last N messages, drop oldest

**Pros:**
- Trivial implementation
- Zero LLM calls
- Fast

**Cons:**
- Loses all old context
- No semantic compression
- Can lose critical information

**Reversibility:** High

#### Option B: LLM-Based Progressive Summarization
**Summary:** Use LLM to summarize chunks of old messages

**Pros:**
- High semantic fidelity
- Preserves key information
- Leverages existing LLM

**Cons:**
- Requires LLM calls (latency, cost)
- Potential summarization errors

**Reversibility:** Medium (can save original messages)

**Algorithm:**
1. When context hits 80% of max tokens:
   - Take messages [0:N/2] (oldest half)
   - Send to LLM: "Summarize this conversation, preserving key decisions and context"
   - Replace messages with single summary message
2. Continue until under threshold

#### Option C: Hybrid (Importance-Weighted Compression)
**Summary:** Classify messages by importance, apply different compression levels

**Pros:**
- Best balance of quality and efficiency
- Preserves critical messages
- Reduces LLM summarization calls

**Cons:**
- Complex importance scoring logic
- More implementation effort

**Reversibility:** High (importance weights are heuristic-based)

**Classification:**
- **Critical** (keep verbatim): User requests, tool results, errors
- **Important** (keep, compress if needed): LLM reasoning, plans
- **Ephemeral** (compress aggressively): "thinking" content, intermediate steps

### Selected Option: C (Hybrid Importance-Weighted Compression)

**Rationale:**
- Aligns with SOLID principles (Strategy pattern)
- Offers best quality-efficiency tradeoff
- Extensible: can add LLM-based summarization later
- Respects KISS for v1 (heuristic-based), allows DRY evolution

**Expected Effects:**
- Context overflow prevention: 95%+ success rate
- Semantic preservation: >90% (estimated)
- Performance: <50ms compression overhead
- Zero emergency truncations

### Implementation Plan

**1. Create Compression Interface** (`internal/core/history/compress.go`)
```go
type Compressor interface {
    Compress(messages []Message, targetTokens int) ([]Message, error)
    Name() string
}

type HybridCompressor struct {
    classifier *MessageClassifier
    summarizer Summarizer  // Interface for future LLM-based summarization
}

type MessageImportance int
const (
    ImportanceCritical MessageImportance = 3
    ImportanceHigh     MessageImportance = 2
    ImportanceMedium   MessageImportance = 1
    ImportanceLow      MessageImportance = 0
)

func (c *MessageClassifier) Classify(msg Message) MessageImportance
```

**2. Implement Heuristic Classifier**
```go
func (c *MessageClassifier) Classify(msg Message) MessageImportance {
    // Critical: User messages, tool results, errors
    if msg.Role == "user" {
        return ImportanceCritical
    }
    if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
        return ImportanceCritical
    }

    // High: Assistant reasoning about code changes
    if strings.Contains(msg.Content, "```") {
        return ImportanceHigh
    }

    // Medium: Regular assistant responses
    if msg.Role == "assistant" {
        return ImportanceMedium
    }

    // Low: Thinking content
    return ImportanceLow
}
```

**3. Compression Algorithm**
```go
func (h *HybridCompressor) Compress(messages []Message, targetTokens int) ([]Message, error) {
    classified := make([]classifiedMessage, len(messages))
    for i, msg := range messages {
        classified[i] = classifiedMessage{
            msg:        msg,
            importance: h.classifier.Classify(msg),
            tokens:     h.tokenizer.Count(msg.Content),
        }
    }

    // Sort by importance (keep critical)
    sort.SliceStable(classified, func(i, j int) bool {
        return classified[i].importance > classified[j].importance
    })

    // Greedy selection within budget
    result := []Message{}
    tokensUsed := 0
    for _, cm := range classified {
        if tokensUsed + cm.tokens <= targetTokens {
            result = append(result, cm.msg)
            tokensUsed += cm.tokens
        }
    }

    // Restore chronological order
    sort.SliceStable(result, func(i, j int) bool {
        return result[i].Timestamp.Before(result[j].Timestamp)
    })

    return result, nil
}
```

**4. Integrate with History**
```go
// internal/core/history.go
func (h *History) Add(msg Message) error {
    h.messages = append(h.messages, msg)
    h.tokensUsed += h.tokenizer.Count(msg.Content)

    // Auto-compress at 80% capacity
    threshold := int(float64(h.maxTokens) * 0.8)
    if h.tokensUsed > threshold {
        compressed, err := h.compressor.Compress(h.messages, h.maxTokens)
        if err != nil {
            return err
        }
        h.messages = compressed
        h.recalculateTokens()

        // Emit compression event
        h.emitter.Emit(Event{
            Type: EventInfo,
            Data: SystemEventData{
                Level:   "info",
                Message: "Context history compressed",
                Details: fmt.Sprintf("Reduced from %d to %d messages", len(h.messages), len(compressed)),
            },
        })
    }

    return nil
}
```

**5. Future LLM-Based Summarization**
```go
type LLMSummarizer struct {
    llm llm.Provider
}

func (s *LLMSummarizer) Summarize(messages []Message) (Message, error) {
    prompt := buildSummarizationPrompt(messages)
    resp, err := s.llm.Complete(ctx, llm.CompletionRequest{
        Messages: []llm.Message{{Role: "user", Content: prompt}},
        Temperature: 0.3,  // Lower temperature for factual summarization
        MaxTokens: 500,
    })
    if err != nil {
        return Message{}, err
    }

    return Message{
        Role:      "assistant",
        Content:   resp.Content,
        Metadata:  map[string]interface{}{"summarized": true, "original_count": len(messages)},
        Timestamp: time.Now(),
    }, nil
}
```

**6. Configuration**
```yaml
# config.yaml
context:
  compression:
    enabled: true
    threshold: 0.8  # Compress at 80% capacity
    strategy: "hybrid"  # Options: "hybrid", "sliding_window", "llm_summary"
    preserve_critical: true
```

### Testing Plan

**Unit Tests:**
- `compress_test.go`: Classifier accuracy, compression algorithm
- `history_compress_test.go`: Integration with history

**Benchmarks:**
- Compression overhead: <50ms for 1000 messages
- Token recalculation: <10ms

**E2E Tests:**
- Long conversation (200+ turns) without overflow
- Verify critical messages (user requests, errors) always preserved

### Verification Plan

**Metrics:**
- Compression ratio: messages before/after
- Token reduction: tokens saved
- Semantic loss: User-reported issues with lost context

**Acceptance Criteria:**
- Zero emergency truncations in 100-turn conversations
- Critical messages: 100% retention
- Compression overhead: <100ms

### Learnings

**New Reusable Patterns:**
- Importance-weighted message selection
- Pluggable compression strategies (Strategy pattern)
- Event-driven compression notifications

---

## Feature 3: VRAM Auto-Tuning

### Problem Statement

Local LLM execution (via Ollama, LM Studio) requires VRAM allocation. Current behavior:
- Users manually configure model parameters
- No automatic detection of VRAM capacity
- Poor UX when model doesn't fit
- Suboptimal quantization choices

Need automatic parameter tuning based on available VRAM to maximize model performance while ensuring successful loading.

### Research Findings

#### State-of-the-Art (2024-2025)

**Ollama:**
- Automatically unloads models from GPU when idle
- Supports hybrid CPU/GPU mode (split layers)
- K/V cache quantization (8-bit reduces VRAM by 50%)
- GGUF quantization: q4_0 (4-bit), q8_0 (8-bit), f16 (unquantized)

**LM Studio:**
- "Likely too large" warnings in UI
- User-selectable quantization levels

**Key Techniques:**
1. **K/V Cache Quantization** (Major 2024 Development)
   - 8-bit quantization: ~50% VRAM reduction
   - Minimal quality loss (q8_0 ≈ f16)
   - Doubles usable context length

2. **Interactive VRAM Estimation**
   - Formula: `VRAM = (param_size * quantization_factor) + (context_length * kv_cache_factor)`
   - Tools: VRAM calculators available online

3. **Flash Attention**
   - Reduces VRAM usage
   - Increases computation speed

4. **Layer Splitting**
   - Offload some layers to CPU
   - Effective VRAM expansion

### Current Codebase Analysis

**LLM Provider System:**
```
internal/llm/
├── provider.go      # Provider interface
├── factory/         # Provider factory
├── ollama/          # Ollama provider
├── lmstudio/        # LMStudio provider
└── openai/          # OpenAI provider
```

**Provider Interface:**
```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    Models(ctx context.Context) ([]Model, error)
    Capabilities() Capabilities
    Name() string
    Close() error
}
```

**Configuration:**
```go
// internal/llm/factory/factory.go
type ProviderConfig struct {
    Type     string
    Model    string
    BaseURL  string
    APIKey   string
    KeyName  string
    // Missing: Quantization, NumGPU, NumThread, etc.
}
```

**Ollama-Specific APIs:**
- `/api/tags`: List models
- `/api/show`: Model details (size, parameters)
- `/api/generate`: Generate completion (with options)

**Ollama Options:**
```json
{
  "num_gpu": 1,
  "num_thread": 8,
  "num_ctx": 2048,
  "num_batch": 512
}
```

### TRIZ Analysis

#### System Function
**Input:** Available VRAM → **Function:** Optimize model parameters → **Output:** Best-fit configuration

#### Main Contradiction
**Improve** model performance (larger models, more context) **without worsening** VRAM constraints

#### Ideal Final Result (IFR)
System automatically detects VRAM, selects optimal quantization and context size, with zero manual tuning.

#### TRIZ Principles Applied

1. **Feedback** → Self-configuring system
   - Query VRAM capacity
   - Adjust parameters dynamically

2. **Parameter Change** → Dynamic quantization selection
   - High VRAM: Use f16 or q8_0
   - Medium VRAM: Use q8_0
   - Low VRAM: Use q4_0

3. **Segmentation** → Modular parameter tuning
   - Separate tuning for: quantization, context length, batch size, GPU layers

4. **Prior Action** → Preemptive capacity check
   - Validate before loading model
   - Warn if likely to fail

5. **Universality** → Provider-agnostic interface
   - Works with Ollama, LM Studio, vLLM

6. **Dynamization** → Adaptive configuration
   - Monitor VRAM during runtime
   - Adjust context length if needed

### Design Options

#### Option A: Static VRAM Profiles
**Summary:** Pre-defined profiles (8GB, 16GB, 24GB, 48GB)

**Pros:**
- Simple implementation
- No runtime probing
- Predictable behavior

**Cons:**
- Not adaptive to actual hardware
- Misses edge cases (shared GPU, containers)

**Reversibility:** High

#### Option B: Runtime VRAM Detection + Calculation
**Summary:** Detect VRAM, calculate max model size, select quantization

**Pros:**
- Accurate to actual hardware
- Adapts to system state
- Professional UX (like LM Studio warnings)

**Cons:**
- Requires VRAM detection (platform-specific)
- More complex

**Reversibility:** Medium

**Algorithm:**
```
1. Detect available VRAM (nvidia-smi, rocm-smi, Metal API)
2. Query model size from provider
3. Calculate required VRAM:
   VRAM_needed = (param_count * bytes_per_param) + (context_len * kv_cache_size)
4. If VRAM_needed > VRAM_available:
   - Try q8_0 quantization
   - If still too large, try q4_0
   - If still too large, reduce context length
   - If still too large, warn user
5. Apply selected parameters
```

#### Option C: Ollama Native Auto-Tuning
**Summary:** Let Ollama handle tuning (it already does some)

**Pros:**
- Zero implementation effort
- Leverages Ollama's intelligence

**Cons:**
- Only works for Ollama
- No control or visibility
- Doesn't help LM Studio, vLLM

**Reversibility:** High

### Selected Option: B (Runtime VRAM Detection + Calculation)

**Rationale:**
- Aligns with professional tools (LM Studio)
- Vendor-agnostic (works across providers)
- Provides user transparency
- Extensible: can add ML-based tuning later

**Expected Effects:**
- Zero model loading failures due to VRAM
- Optimal quantization selection: 90%+ accuracy
- User satisfaction: High (no manual tuning)

### Implementation Plan

**1. Create VRAM Detection Module** (`internal/llm/vram/`)
```go
package vram

type Detector interface {
    TotalVRAM() (int64, error)      // Bytes
    AvailableVRAM() (int64, error)  // Bytes
    GPUName() (string, error)
}

// Platform-specific implementations
type NvidiaDetector struct{}  // Uses nvidia-smi
type AMDDetector struct{}     // Uses rocm-smi
type MetalDetector struct{}   // Uses Metal API (macOS)
type CPUFallback struct{}     // Returns 0 (CPU-only)

func NewDetector() Detector {
    // Auto-detect platform
    if hasNVIDIA() {
        return &NvidiaDetector{}
    } else if hasAMD() {
        return &AMDDetector{}
    } else if hasMetal() {
        return &MetalDetector{}
    }
    return &CPUFallback{}
}
```

**2. Implement NVIDIA Detection**
```go
func (d *NvidiaDetector) AvailableVRAM() (int64, error) {
    cmd := exec.Command("nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits")
    output, err := cmd.Output()
    if err != nil {
        return 0, err
    }

    // Parse output: "15000" (MiB)
    vramMiB, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
    if err != nil {
        return 0, err
    }

    return vramMiB * 1024 * 1024, nil  // Convert to bytes
}
```

**3. Create Model Size Calculator** (`internal/llm/vram/calculator.go`)
```go
type Calculator struct {
    detector Detector
}

type ModelRequirements struct {
    MinVRAM         int64
    RecommendedVRAM int64
    Quantization    string  // "f16", "q8_0", "q4_0"
    ContextLength   int
    NumGPULayers    int
}

func (c *Calculator) Calculate(modelSize int64, contextLen int) (*ModelRequirements, error) {
    availableVRAM, err := c.detector.AvailableVRAM()
    if err != nil {
        return nil, err
    }

    // Quantization factors
    quantFactors := map[string]float64{
        "f16":  1.0,
        "q8_0": 0.5,
        "q4_0": 0.25,
    }

    // KV cache estimation: ~1 byte per token per layer (simplified)
    kvCacheSize := int64(contextLen * 32 * 2)  // 32 layers, 2 bytes per token

    // Try quantizations in order of quality
    for _, quant := range []string{"f16", "q8_0", "q4_0"} {
        factor := quantFactors[quant]
        modelVRAM := int64(float64(modelSize) * factor)
        totalVRAM := modelVRAM + kvCacheSize

        if totalVRAM <= availableVRAM {
            return &ModelRequirements{
                MinVRAM:         totalVRAM,
                RecommendedVRAM: totalVRAM + (1024 * 1024 * 1024),  // +1GB headroom
                Quantization:    quant,
                ContextLength:   contextLen,
                NumGPULayers:    -1,  // Auto (all layers)
            }, nil
        }
    }

    // Fallback: Reduce context length
    if contextLen > 2048 {
        return c.Calculate(modelSize, 2048)
    }

    // Fallback: Use CPU offloading
    return &ModelRequirements{
        MinVRAM:         0,
        RecommendedVRAM: availableVRAM,
        Quantization:    "q4_0",
        ContextLength:   2048,
        NumGPULayers:    16,  // Only some layers on GPU
    }, nil
}
```

**4. Integrate with Ollama Provider** (`internal/llm/ollama/provider.go`)
```go
func (p *OllamaProvider) AutoTune(ctx context.Context) error {
    // Get model info
    info, err := p.getModelInfo(ctx, p.model)
    if err != nil {
        return err
    }

    // Calculate optimal parameters
    calc := vram.NewCalculator(vram.NewDetector())
    reqs, err := calc.Calculate(info.Size, p.config.ContextLength)
    if err != nil {
        return err
    }

    // Apply parameters
    p.options = map[string]interface{}{
        "num_ctx":      reqs.ContextLength,
        "num_gpu":      reqs.NumGPULayers,
        // Ollama handles quantization via model name: e.g., "llama2:7b-q4_0"
    }

    log.Printf("Auto-tuned for %s: quantization=%s, context=%d, gpu_layers=%d",
        p.model, reqs.Quantization, reqs.ContextLength, reqs.NumGPULayers)

    return nil
}
```

**5. Add Configuration**
```yaml
# config.yaml
llm:
  auto_tune: true  # Enable VRAM auto-tuning
  vram:
    detect: true
    headroom: 1024  # MiB of VRAM to reserve for system
```

**6. User Warnings**
```go
func (p *OllamaProvider) validateModel(ctx context.Context) error {
    calc := vram.NewCalculator(vram.NewDetector())
    info, _ := p.getModelInfo(ctx, p.model)

    reqs, err := calc.Calculate(info.Size, p.config.ContextLength)
    if err != nil {
        return err
    }

    availableVRAM, _ := calc.detector.AvailableVRAM()
    if reqs.MinVRAM > availableVRAM {
        return fmt.Errorf(
            "model too large for available VRAM: needs %s, have %s. Try a smaller model or quantization.",
            humanize.Bytes(uint64(reqs.MinVRAM)),
            humanize.Bytes(uint64(availableVRAM)),
        )
    }

    return nil
}
```

### Testing Plan

**Unit Tests:**
- `vram_detector_test.go`: Mock nvidia-smi output, test parsing
- `calculator_test.go`: Quantization selection logic

**Integration Tests:**
- `ollama_autotune_test.go`: End-to-end auto-tuning

**Manual Tests:**
- Test on various hardware: NVIDIA RTX 3090 (24GB), RTX 4060 (8GB), CPU-only
- Verify model loading succeeds with auto-tuned parameters

### Verification Plan

**Metrics:**
- Model loading success rate: 100%
- Quantization accuracy: "best-fit" selected >90% of time
- Performance: VRAM detection <500ms

**Acceptance Criteria:**
- Zero "out of VRAM" errors with auto-tuning enabled
- User warnings shown when model won't fit
- Graceful degradation (CPU offloading) when VRAM insufficient

### Learnings

**New Reusable Patterns:**
- Platform-agnostic hardware detection
- Resource-constrained optimization
- Graceful degradation strategies

---

## Feature 4: Cycle Auto-Discovery

### Problem Statement

Autonomous agents can enter infinite reasoning loops where:
- LLM repeats similar responses
- Same tool called repeatedly with no progress
- Agent gets "stuck" without external intervention

Need automatic detection and intervention to maintain reliability.

### Research Findings

#### Industry Problems (2024-2025)

**LLM Infinite Loops:**
- "Texts differing by just a few words can trigger infinite loops"
- "Potentially infinite output with million-dollar bills"
- Source: GDELT Project on LLM entity extraction

**Multi-Agent Loop Issues:**
- Supervisor agents stuck calling same agent repeatedly
- Route function errors cause endless cycles
- Source: LangChain discussions #17872

**Loop Detection Strategies:**
1. **Maximum Iteration Limits**
   - Common safeguard
   - LoopAgent terminates after N iterations

2. **State Comparison**
   - Compare consecutive outputs for similarity
   - Detect repeated patterns

3. **Termination Conditions**
   - Explicit success/failure states
   - Condition evaluation each iteration

**Agent Loop Design:**
- Mental model: Perception → Planning → Action → repeat
- Critical skill: Designing effective termination

### Current Codebase Analysis

**Agent Loop:**
```go
// internal/core/agent.go
func (a *Agent) ProcessRequest(ctx context.Context, req *AgentRequest) (<-chan Event, error) {
    for turn := 0; turn < a.config.MaxTurns; turn++ {
        // Get LLM response
        resp, err := a.llm.Complete(ctx, llmReq)

        // Execute tools
        for _, toolCall := range resp.ToolCalls {
            result, err := a.executor.Execute(ctx, toolCall)
            // ...
        }

        // Check if done
        if resp.StopReason == "end_turn" {
            break
        }
    }
}
```

**Existing Safeguards:**
- `MaxTurns` limit (default: 50)
- `Timeout` (default: 5 minutes)

**Missing:**
- No similarity detection
- No pattern recognition
- No automatic intervention

### TRIZ Analysis

#### System Function
**Input:** Agent conversation history → **Function:** Detect cycles → **Output:** Intervention signal

#### Main Contradiction
**Improve** autonomous operation **without worsening** risk of infinite loops

#### Ideal Final Result (IFR)
Agent automatically detects when stuck in cycle, intervenes to break loop, continues productively.

#### TRIZ Principles Applied

1. **Feedback** → Self-monitoring system
   - Track response similarity over time
   - Detect anomalies

2. **Prior Action** → Preventive measures
   - Detect early warning signs (similarity increasing)
   - Intervene before critical state

3. **Parameter Change** → Adaptive thresholds
   - Strict detection for short conversations
   - Relaxed for long conversations (similarity expected)

4. **Taking Out** → Externalize cycle detection
   - Separate component: `CycleDetector`
   - Doesn't pollute agent core logic

5. **Dynamization** → Adaptive intervention
   - Soft intervention: Inject "reflection" prompt
   - Hard intervention: Force context re-summarization
   - Nuclear intervention: Terminate turn, ask user

6. **Copying** → Snapshot comparison
   - Store response snapshots
   - Compare against history

### Design Options

#### Option A: Simple Similarity Threshold
**Summary:** Compare last 3 responses, if >80% similar, trigger intervention

**Pros:**
- Simple implementation
- Fast (O(1) comparison)

**Cons:**
- False positives (legitimately similar responses)
- Naive (doesn't detect semantic similarity)

**Reversibility:** High

**Algorithm:**
```go
if similarity(response[n-1], response[n-2]) > 0.8 &&
   similarity(response[n-2], response[n-3]) > 0.8 {
    // Cycle detected
    triggerIntervention()
}
```

#### Option B: Embedding-Based Semantic Similarity
**Summary:** Compute embeddings of responses, use cosine similarity

**Pros:**
- Detects semantic similarity (not just string matching)
- Lower false positive rate

**Cons:**
- Requires embedding model
- Higher computational cost

**Reversibility:** High

#### Option C: Pattern-Based Detection
**Summary:** Detect specific patterns (repeated tool calls, oscillating states)

**Pros:**
- Highly accurate for known patterns
- No false positives for those patterns

**Cons:**
- Misses novel cycle types
- Requires manual pattern definition

**Reversibility:** High

**Patterns:**
- Same tool called 3+ times in a row with no state change
- Response oscillation: A → B → A → B
- Error loop: Same error 3+ times

### Selected Option: Hybrid (A + C)

**Rationale:**
- Combines simplicity of string similarity with pattern detection
- Covers common cases (repeated tool calls, oscillation)
- Extensible: Can add embedding-based detection later
- KISS for v1, but effective

**Expected Effects:**
- Cycle detection accuracy: >85%
- False positive rate: <5%
- Intervention success: >70% (agent recovers)

### Implementation Plan

**1. Create CycleDetector Component** (`internal/core/cycle/detector.go`)
```go
package cycle

type Detector struct {
    history []Snapshot
    config  Config
}

type Snapshot struct {
    Turn         int
    Response     string
    ToolCalls    []string
    Timestamp    time.Time
}

type Config struct {
    WindowSize        int     // Number of snapshots to compare (default: 3)
    SimilarityThresh  float64 // Similarity threshold (default: 0.8)
    ToolRepeatLimit   int     // Max identical tool calls (default: 3)
}

type CycleType int
const (
    CycleNone CycleType = iota
    CycleSimilarResponses
    CycleRepeatedTool
    CycleOscillation
    CycleSameError
)

func (d *Detector) Check() (CycleType, error)
```

**2. Implement Similarity Detection**
```go
func (d *Detector) detectSimilarResponses() bool {
    if len(d.history) < d.config.WindowSize {
        return false
    }

    recent := d.history[len(d.history)-d.config.WindowSize:]

    // Compare consecutive pairs
    for i := 1; i < len(recent); i++ {
        sim := similarity(recent[i-1].Response, recent[i].Response)
        if sim < d.config.SimilarityThresh {
            return false  // Not all similar
        }
    }

    return true  // All consecutive pairs are similar
}

func similarity(a, b string) float64 {
    // Jaccard similarity (simple but effective)
    wordsA := strings.Fields(strings.ToLower(a))
    wordsB := strings.Fields(strings.ToLower(b))

    setA := make(map[string]bool)
    for _, w := range wordsA {
        setA[w] = true
    }

    setB := make(map[string]bool)
    for _, w := range wordsB {
        setB[w] = true
    }

    intersection := 0
    for w := range setA {
        if setB[w] {
            intersection++
        }
    }

    union := len(setA) + len(setB) - intersection
    if union == 0 {
        return 1.0
    }

    return float64(intersection) / float64(union)
}
```

**3. Implement Pattern Detection**
```go
func (d *Detector) detectRepeatedTool() bool {
    if len(d.history) < d.config.ToolRepeatLimit {
        return false
    }

    recent := d.history[len(d.history)-d.config.ToolRepeatLimit:]

    // Check if all recent snapshots called the same tool
    if len(recent[0].ToolCalls) == 0 {
        return false
    }

    firstTool := recent[0].ToolCalls[0]
    for i := 1; i < len(recent); i++ {
        if len(recent[i].ToolCalls) == 0 || recent[i].ToolCalls[0] != firstTool {
            return false
        }
    }

    return true
}

func (d *Detector) detectOscillation() bool {
    if len(d.history) < 4 {
        return false
    }

    recent := d.history[len(d.history)-4:]

    // Check A → B → A → B pattern
    if similarity(recent[0].Response, recent[2].Response) > 0.9 &&
       similarity(recent[1].Response, recent[3].Response) > 0.9 &&
       similarity(recent[0].Response, recent[1].Response) < 0.5 {
        return true
    }

    return false
}
```

**4. Implement Interventions** (`internal/core/cycle/intervention.go`)
```go
type Intervention interface {
    Apply(ctx context.Context, history []Message) ([]Message, error)
    Name() string
}

// Soft: Inject reflection prompt
type ReflectionIntervention struct{}
func (i *ReflectionIntervention) Apply(ctx context.Context, history []Message) ([]Message, error) {
    reflectionPrompt := Message{
        Role: "user",
        Content: "I notice you may be repeating yourself. Let's take a step back. " +
                 "What is the core issue? Is there a different approach we should try?",
        Timestamp: time.Now(),
    }
    return append(history, reflectionPrompt), nil
}

// Medium: Force context summarization
type SummarizeIntervention struct {
    compressor *history.Compressor
}
func (i *SummarizeIntervention) Apply(ctx context.Context, history []Message) ([]Message, error) {
    compressed, err := i.compressor.Compress(history, len(history)/2)  // Compress to 50%
    if err != nil {
        return history, err
    }

    notice := Message{
        Role:    "system",
        Content: "Context has been summarized to help you focus on the current task.",
        Timestamp: time.Now(),
    }

    return append(compressed, notice), nil
}

// Hard: Escalate to user
type UserEscalationIntervention struct {
    emitter *EventEmitter
}
func (i *UserEscalationIntervention) Apply(ctx context.Context, history []Message) ([]Message, error) {
    i.emitter.Emit(Event{
        Type: EventTurnPaused,
        Data: TurnEventData{
            Status:  "paused",
            Message: "Agent appears stuck. Please provide guidance or restart.",
        },
    })

    // Return history unmodified, wait for user input
    return history, nil
}
```

**5. Integrate with Agent** (`internal/core/agent.go`)
```go
func (a *Agent) ProcessRequest(ctx context.Context, req *AgentRequest) (<-chan Event, error) {
    detector := cycle.NewDetector(cycle.Config{
        WindowSize:       3,
        SimilarityThresh: 0.8,
        ToolRepeatLimit:  3,
    })

    for turn := 0; turn < a.config.MaxTurns; turn++ {
        // Get LLM response
        resp, err := a.llm.Complete(ctx, llmReq)

        // Record snapshot
        detector.Record(cycle.Snapshot{
            Turn:      turn,
            Response:  resp.Content,
            ToolCalls: extractToolNames(resp.ToolCalls),
            Timestamp: time.Now(),
        })

        // Check for cycles
        cycleType, err := detector.Check()
        if cycleType != cycle.CycleNone {
            // Apply intervention based on severity
            intervention := a.selectIntervention(cycleType, turn)
            history, err = intervention.Apply(ctx, history)
            if err != nil {
                return nil, err
            }

            // Emit event
            a.emitter.Emit(Event{
                Type: EventWarning,
                Data: SystemEventData{
                    Level:   "warning",
                    Message: fmt.Sprintf("Cycle detected: %s. Applied intervention: %s", cycleType, intervention.Name()),
                },
            })

            // If user escalation, pause
            if _, ok := intervention.(*cycle.UserEscalationIntervention); ok {
                return events, nil  // Pause here
            }
        }

        // ... rest of turn processing
    }
}

func (a *Agent) selectIntervention(cycleType cycle.CycleType, turn int) cycle.Intervention {
    // Escalation ladder
    if turn < 10 {
        return &cycle.ReflectionIntervention{}  // Soft
    } else if turn < 30 {
        return &cycle.SummarizeIntervention{compressor: a.compressor}  // Medium
    } else {
        return &cycle.UserEscalationIntervention{emitter: a.emitter}  // Hard
    }
}
```

**6. Configuration**
```yaml
# config.yaml
agent:
  cycle_detection:
    enabled: true
    window_size: 3
    similarity_threshold: 0.8
    tool_repeat_limit: 3
```

### Testing Plan

**Unit Tests:**
- `detector_test.go`: Similarity calculation, pattern detection
- `intervention_test.go`: Each intervention type

**Integration Tests:**
- `cycle_integration_test.go`: Synthetic cycle scenarios
  - Create conversation with 3 identical responses
  - Verify cycle detected
  - Verify intervention applied

**E2E Tests:**
- Manual testing: Prompt agent to repeat itself
- Verify intervention triggers

### Verification Plan

**Metrics:**
- Cycle detection rate: % of actual cycles detected
- False positive rate: % of non-cycles flagged
- Intervention success rate: % of cycles broken

**Acceptance Criteria:**
- Detect >80% of repetitive cycles
- False positives: <5%
- Intervention breaks cycle: >70% of time

### Learnings

**New Reusable Patterns:**
- State snapshot comparison for anomaly detection
- Escalation ladder for interventions
- Pattern-based behavior monitoring

---

## Feature 5: Enhanced Approval Mechanisms

### Problem Statement

Current approval system (from codebase):
- `ApprovalHandler` function-based
- Blocks execution until user responds
- Basic approve/deny/modify flow

Needs:
- UI integration (TUI approval prompts)
- Persistent approval policies (remember decisions)
- Approval history/audit trail
- Fine-grained control (per-command patterns)

### Research Findings

#### Industry Solutions (2024-2025)

**LangGraph:**
- `interrupt()` function pauses graph mid-execution
- Wait for human input, resume cleanly
- Top choice for HITL checkpoints

**Amazon Bedrock Agents:**
- User confirmation: Boolean validation
- Return of control (ROC): More complex workflows
- Structured approval requests

**OpenAI Agents SDK:**
- `needsApproval` option on tools
- Async function returning boolean
- Agent interrupts for approval

**Design Patterns:**
- Approve/reject checkpoints before critical steps
- Review and modify before execution
- Conditional auto-approval based on policies

### Current Codebase Analysis

**Existing Approval System:**
```go
// internal/core/agent.go
type ApprovalRequest struct {
    ID              string
    Command         *Command
    Reason          string
    WorkDir         string
    Timestamp       time.Time
}

type ApprovalResponse struct {
    RequestID       string
    Approved        bool
    Reason          string
    ModifiedCommand string
    Timestamp       time.Time
}

type ApprovalHandler func(ApprovalRequest) ApprovalResponse
```

**Integration Points:**
- Agent calls `approvalHandler(req)` when dangerous command detected
- Blocks until response received
- Timeout: `config.ApprovalTimeout` (default 60s)

**Events:**
- `EventCommandApproval`: Request sent
- `EventCommandApproved`: Approved
- `EventCommandDenied`: Denied

**Missing:**
- No TUI approval dialog
- No policy persistence
- No audit trail

### TRIZ Analysis

#### System Function
**Input:** Dangerous command → **Function:** Get user approval → **Output:** Approved/denied decision

#### Main Contradiction
**Improve** user control and safety **without worsening** agent autonomy and UX friction

#### Ideal Final Result (IFR)
User approves commands seamlessly via TUI, policies auto-approve trusted patterns, full audit trail maintained.

#### TRIZ Principles Applied

1. **Prior Action** → Preemptive policies
   - User defines approval policies in advance
   - Auto-approve safe patterns

2. **Segmentation** → Layered approval
   - Level 1: Auto-approve (based on policy)
   - Level 2: User approval (interactive)
   - Level 3: Always deny (blacklist)

3. **Universality** → Reusable policy engine
   - Same engine for CLI, TUI, API modes

4. **Feedback** → Learning system
   - Track approval history
   - Suggest policy updates

5. **Taking Out** → Externalize policies
   - Store in config file
   - Editable outside runtime

6. **Copying** → Shadow execution (future)
   - Simulate command without executing
   - Show user predicted outcome

### Design Options

#### Option A: Minimal TUI Approval Dialog
**Summary:** Simple modal prompt when approval needed

**Pros:**
- Simple implementation
- Immediate user feedback

**Cons:**
- No policy support
- No history

**Reversibility:** High

**UI:**
```
┌─────────────────────────────────────────────────┐
│ Approval Required                               │
├─────────────────────────────────────────────────┤
│ Command: rm -rf /tmp/build                      │
│ Reason:  Destructive file operation             │
│ WorkDir: /home/user/project                     │
│                                                 │
│ [A]pprove  [D]eny  [M]odify  [?]Help            │
└─────────────────────────────────────────────────┘
```

#### Option B: Policy-Aware Approval System
**Summary:** Approval engine with configurable policies

**Pros:**
- Flexible and powerful
- Reduces user friction (auto-approve trusted)
- Audit trail

**Cons:**
- More complex implementation
- Requires policy language

**Reversibility:** Medium

**Policy Example:**
```yaml
approval_policies:
  - pattern: "git commit*"
    action: auto_approve
    reason: "Git commits are safe"

  - pattern: "rm -rf*"
    action: require_approval
    reason: "Destructive operation"

  - pattern: "curl * | bash"
    action: always_deny
    reason: "Potential security risk"
```

#### Option C: Full HITL Framework
**Summary:** Comprehensive human-in-the-loop system with approvals, reviews, modifications

**Pros:**
- Feature-rich
- Matches industry standards (LangGraph)

**Cons:**
- Significant implementation effort
- May be overkill for v1

**Reversibility:** Low

### Selected Option: A (TUI Dialog Only - Leverage Existing Infrastructure)

**IMPORTANT: Existing Infrastructure Discovered**
After codebase analysis, Feature 5 is **95% complete**:
- ✅ `internal/core/validator.go` (855 lines) - Complete classification system
- ✅ `internal/core/executor.go` - Validates before execution
- ✅ `internal/core/agent.go` - ApprovalHandler interface + full approval flow
- ❌ **Only missing: TUI approval dialog (~150 lines)**

**Rationale:**
- Don't rebuild what exists - Validator already has comprehensive patterns
- Focus on missing UI layer only
- Audit logging is optional enhancement

**Expected Effects:**
- Implementation time: 4-5 hours (not 1 week)
- Zero duplicate systems
- Leverages battle-tested Validator patterns

### Implementation Plan

**Note:** Validator/Executor/Agent already handle approval flow. Only UI missing.

**1. Create TUI Approval Dialog** (`internal/ui/overlay/approval.go`) **~150 lines**
```go
package overlay

type ApprovalDialog struct {
    request  core.ApprovalRequest
    response chan core.ApprovalResponse
}

func (d *ApprovalDialog) Render(w io.Writer, width, height int) error {
    // Render modal
    fmt.Fprintf(w, "\033[2J\033[H")  // Clear screen

    box := []string{
        "┌─────────────────────────────────────────────────┐",
        "│ Approval Required                               │",
        "├─────────────────────────────────────────────────┤",
        fmt.Sprintf("│ Command: %-39s │", truncate(d.request.Command.Raw, 39)),
        fmt.Sprintf("│ Reason:  %-39s │", truncate(d.request.Reason, 39)),
        fmt.Sprintf("│ WorkDir: %-39s │", truncate(d.request.WorkDir, 39)),
        "│                                                 │",
        "│ [A]pprove  [D]eny  [M]odify  [?]Help            │",
        "└─────────────────────────────────────────────────┘",
    }

    for _, line := range box {
        fmt.Fprintln(w, line)
    }

    return nil
}

func (d *ApprovalDialog) HandleKey(key keyboard.Key) error {
    switch key.Rune {
    case 'a', 'A':
        d.response <- core.ApprovalResponse{
            RequestID: d.request.ID,
            Approved:  true,
            Timestamp: time.Now(),
        }

    case 'd', 'D':
        d.response <- core.ApprovalResponse{
            RequestID: d.request.ID,
            Approved:  false,
            Reason:    "User denied",
            Timestamp: time.Now(),
        }

    case 'm', 'M':
        // Enter modify mode (future enhancement)
        d.enterModifyMode()
    }

    return nil
}
```

**5. Wire TUI Approval Handler** (`cmd/spin/tui.go`)
```go
func setupApprovalHandler(ui *adapters.PureTTY) core.ApprovalHandler {
    return func(req core.ApprovalRequest) core.ApprovalResponse {
        // Create approval dialog
        dialog := overlay.NewApprovalDialog(req)

        // Show dialog
        responseChan := make(chan core.ApprovalResponse, 1)

        // Render dialog via TUI
        ui.ShowOverlay(dialog, responseChan)

        // Wait for user response or timeout
        select {
        case resp := <-responseChan:
            return resp
        case <-time.After(60 * time.Second):
            return core.ApprovalResponse{
                RequestID: req.ID,
                Approved:  false,
                Reason:    "Timeout",
                Timestamp: time.Now(),
            }
        }
    }
}
```

**6. Configuration**
```yaml
# ~/.spin/config.yaml
security:
  approval:
    enabled: true
    timeout: 60s
    audit_log: ~/.spin/approval_audit.jsonl

    policies:
      - pattern: "git status"
        action: auto_approve
        reason: "Read-only git command"

      - pattern: "git commit*"
        action: auto_approve
        reason: "Safe git operation"

      - pattern: "make test"
        action: auto_approve
        reason: "Test execution is safe"

      - pattern: "rm -rf*"
        action: require_approval
        reason: "Destructive file operation"

      - pattern: "curl * | bash"
        action: always_deny
        reason: "Security risk: arbitrary code execution"

      - pattern: "sudo*"
        action: always_deny
        reason: "Elevated privileges not allowed"
```

### Testing Plan

**Unit Tests:**
- `policy_test.go`: Pattern matching, action selection
- `audit_test.go`: Log writing, parsing

**Integration Tests:**
- `approval_integration_test.go`: Policy evaluation + audit logging
- Mock approval handler, verify policy flow

**E2E Tests:**
- Manual TUI testing: Trigger approval dialog
- Verify UI rendering, keyboard input

### Verification Plan

**Metrics:**
- Policy match rate: % of commands matched by policy
- Auto-approval rate: % of commands auto-approved
- User approval response time: Median time to approve/deny

**Acceptance Criteria:**
- Policies loaded from config: 100%
- Audit log written: 100% of approval events
- TUI dialog renders correctly: Visual validation

### Learnings

**New Reusable Patterns:**
- Policy engine for behavioral control
- Audit trail for compliance
- Modal dialogs in TUI

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
**Objectives:** Core infrastructure for new features

**Tasks:**
1. Create package structure:
   - `internal/ui/statusbar/`
   - `internal/core/history/compress/`
   - `internal/llm/vram/`
   - `internal/core/cycle/`
   - ~~`internal/security/approval/`~~ ❌ **NOT NEEDED** - Validator/Executor exist

2. Implement basic interfaces
3. Write unit tests (target: 90%+ coverage)

**Deliverables:**
- Package scaffolding
- Interface definitions
- Unit tests passing

---

### Phase 2: Status Bar (Week 2)
**Objectives:** Functional status bar with real-time metrics

**Tasks:**
1. Implement `StatusBar` component
2. Integrate with `Coordinator`
3. Wire event subscriptions
4. Add terminal width adaptation

**Deliverables:**
- Working status bar in TUI
- E2E test with live metrics
- Documentation

**Definition of Done:**
- Status bar displays: agent state, context %, provider, tokens/sec
- Updates in real-time from events
- Works in terminals 60-200 columns wide
- Zero scrollback disruption

---

### Phase 3: Context Summarization (Week 3)
**Objectives:** Automatic context compression

**Tasks:**
1. Implement `MessageClassifier`
2. Implement `HybridCompressor`
3. Integrate with `History.Add()`
4. Add compression event emission

**Deliverables:**
- Automatic compression at 80% capacity
- Importance-weighted message selection
- Compression events in TUI

**Definition of Done:**
- Zero emergency truncations in 100-turn conversations
- Critical messages: 100% retention
- Compression overhead: <100ms
- Tests: 90%+ coverage

---

### Phase 4: VRAM Auto-Tuning (Week 4)
**Objectives:** Intelligent model parameter selection

**Tasks:**
1. Implement VRAM detectors (NVIDIA, AMD, Metal)
2. Implement `Calculator`
3. Integrate with Ollama provider
4. Add user warnings for too-large models

**Deliverables:**
- Auto-tuned quantization and context length
- Platform-specific VRAM detection
- User warnings in TUI

**Definition of Done:**
- Model loading success: 100% with auto-tuning
- Quantization selection: "best-fit" >90% of time
- VRAM detection: <500ms
- Tests: 90%+ coverage

---

### Phase 5: Cycle Auto-Discovery (Week 5)
**Objectives:** Detect and break reasoning loops

**Tasks:**
1. Implement `CycleDetector`
2. Implement interventions (Reflection, Summarize, Escalate)
3. Integrate with `Agent.ProcessRequest()`
4. Add cycle warning events

**Deliverables:**
- Automatic cycle detection
- Multi-level interventions
- User escalation when stuck

**Definition of Done:**
- Cycle detection: >80% of actual cycles
- False positives: <5%
- Intervention breaks cycle: >70% of time
- Tests: 90%+ coverage

---

### Phase 6: Enhanced Approval (Week 6) - CORRECTED SCOPE
**Objectives:** TUI approval dialog only (validation already exists)

**Tasks:**
1. ~~Implement `PolicyEngine`~~ ❌ **NOT NEEDED** - Validator already exists
2. Create TUI approval dialog (`internal/ui/overlay/approval.go`) - **~150 lines**
3. Wire dialog to existing ApprovalHandler - **~30 lines**
4. Optional: Add audit logger (`internal/security/audit/`) - **~80 lines**

**Deliverables:**
- TUI approval modal dialog
- Keyboard input handling (A/D keys)
- Optional: Audit trail logging

**Definition of Done:**
- TUI dialog renders when Interactive/Dangerous command detected
- User can approve with 'A', deny with 'D'
- Timeout works (60s default, auto-deny)
- Forbidden commands auto-blocked (no dialog)
- Safe commands auto-executed (no dialog)
- Tests: Modal rendering + key handling

---

### Phase 7: Integration & Polish (Week 7)
**Objectives:** End-to-end integration, documentation, dogfooding

**Tasks:**
1. Integration testing across all features
2. Performance profiling and optimization
3. Documentation updates (docs/, specs/)
4. User guide / tutorial

**Deliverables:**
- Full system integration
- Updated documentation
- Tutorial videos/guides

**Definition of Done:**
- All features working together
- No integration bugs
- Documentation complete
- Manual QA passed

---

### Phase 8: Production Readiness (Week 8)
**Objectives:** Production-grade reliability

**Tasks:**
1. Security review
2. Performance benchmarks
3. Error handling review
4. Production config examples

**Deliverables:**
- Security audit report
- Performance benchmark results
- Production-ready release

**Definition of Done:**
- Security review: No critical issues
- Performance: All SLOs met
- Error handling: Comprehensive
- Ready for production use

---

## References

### Web Research

**TUI & Status Bars:**
- Terminal Trove: TUI Tools (https://terminaltrove.com/categories/tui/)
- Bubble Tea Framework (Go): Modern TUI with MUV architecture
- s-tui: Terminal CPU monitoring with status displays
- "Build a System Monitor TUI in Go" (Ivan Penchev)

**LLM Context Compression:**
- "Extending Context Window via Semantic Compression" (arXiv 2312.09571v1)
- "In-Context Former: Lightning-fast Compressing Context" (arXiv 2406.13618v1)
- "LLMLingua: Innovating LLM efficiency with prompt compression" (Microsoft Research 2024)
- "Recurrent Context Compression" (arXiv 2406.06110v1)
- "Progressive Summarisation" (Anthony Sun, Medium)

**VRAM Optimization:**
- "Using Ollama to Serve Quantized Models" (RunPod.io)
- "Optimizing Ollama VRAM Settings" (Peddals Blog)
- "Bringing K/V Context Quantisation to Ollama" (smcleod.net, Dec 2024)
- "VRAM Calculator for Self-Hosting LLMs" (AIMultiple)
- "Context Kills VRAM: How to Run LLMs on consumer GPUs" (Lyx, Medium)

**Cycle Detection:**
- "LLM Infinite Loops & Failure Modes" (GDELT Project)
- "Designing agentic loops" (Simon Willison)
- "The Intelligent Loop: A Guide to Modern LLM Agents" (DEV Community)
- LangChain Discussion #17872: Multi-agent loop issues

**Approval Mechanisms:**
- "Human-in-the-Loop for AI Agents" (Permit.io Blog 2024)
- "Human in the loop" (OpenAI Agents SDK)
- "Implement human-in-the-loop confirmation" (AWS Bedrock Agents)
- "LangGraph's human-in-the-loop" (LangChain AI)
- "Building an AI Agent with HITL" (Cloudflare Agents SDK)

**TRIZ Methodology:**
- "TRIZ: Theory of Inventive Problem Solving" (Wikipedia)
- "40 Inventive Principles" (TRIZ40.com)
- "TRIZ - A Powerful Methodology for Creative Problem Solving" (MindTools)
- "New product development with TRIZ methodology" (ResearchGate)

**Competitor Analysis:**
- "Claude Code vs Cursor: Complete comparison guide 2025" (Northflank Blog)
- "Cursor vs. Claude Code vs. GitHub Copilot" (DredYson.com)
- "Running Open-Source AI Coding Assistants Aider and Claude Dev" (John Maeda, Medium)
- "Cursor Agent vs. Claude Code" (Haihai.ai)

### Codebase References

**Existing Systems:**
- `internal/ui/`: TUI infrastructure (term, prompt, output, blocks, adapters)
- `internal/core/`: Agent, events, history, tokenizer
- `internal/llm/`: Provider interface, Ollama, LM Studio, tokenizer
- `internal/security/`: Validator, approval handler
- `internal/session/`: Session persistence

**Key Files:**
- `internal/ui/term/tty.go`: ANSI terminal control
- `internal/ui/output/coordinator.go`: Output + prompt coordination
- `internal/core/event.go`: Event system with typed payloads
- `internal/core/history.go`: Conversation history with token tracking
- `internal/core/agent.go`: Agent loop and approval integration
- `internal/llm/ollama/provider.go`: Ollama provider implementation

**Documentation:**
- `docs/tui.md`: TUI architecture and design
- `docs/performance.md`: Performance benchmarks
- `docs/modes.md`: Task modes (regular, review, compact, planning)
- `docs/packages/core.md`: Core package overview
- `docs/packages/llm.md`: LLM provider documentation
- `AGENTS.md`: Development workflow and quality standards

---

## Appendix: TRIZ Inventive Principles Summary

### Applied Principles Across Features

**1. Segmentation** (Modularization)
- Status bar: Separate component from timeline
- Context compression: Multi-level summarization
- Approval: Layered policies

**2. Prior Action** (Caching/Precomputation)
- Context compression: Trigger at 80%, not 100%
- VRAM tuning: Validate before loading
- Approval: Preemptive policies

**3. Feedback** (Observability, Self-Healing)
- Status bar: Event-driven updates
- Cycle detection: Self-monitoring
- VRAM: Self-configuring

**4. Dynamization** (Autoscaling, Adaptive Algorithms)
- Status bar: Adaptive layout
- Context compression: Adaptive ratio
- VRAM: Dynamic quantization
- Cycle detection: Adaptive thresholds

**5. Taking Out** (Externalizing Configs)
- Approval: Externalize policies to config
- Cycle detection: Separate component

**6. Universality** (Generic Components, Reusable APIs)
- Context compression: Pluggable strategies
- VRAM: Provider-agnostic interface
- Approval: Reusable policy engine

**7. Parameter Change** (Feature Toggles, Sampling)
- VRAM: Dynamic quantization selection
- Cycle detection: Adaptive thresholds
- Approval: Policy-based control

**8. Nested Doll** (Containers, Dependency Injection)
- Status bar: Hierarchical metric display
- Context compression: Hierarchical levels
- Approval: Escalation ladder

**9. Local Quality** (Domain-Specific Optimizations)
- Status bar: Optimized rendering
- Context compression: Importance weighting

**10. Intermediary** (Queues, Gateways)
- Approval: Policy engine as intermediary

---

**End of Research Document**

*Generated: 2025-10-12*
*Total Research Time: ~4 hours*
*Methodology: TRIZ-based systematic innovation*
*Next Steps: Begin FRD creation for Phase 1*
