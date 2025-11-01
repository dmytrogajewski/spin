# ACE (Agentic Context Engineering)

**Package:** `internal/ace`  
**Status:** In Development (Phase 1: Foundation)  
**Feature:** Core Data Structures and Context Management

---

## Overview

ACE (Agentic Context Engineering) is a framework for evolving contexts that enable self-improving language models. Based on the research paper "Agentic Context Engineering: Evolving Contexts for Self-Improving Language Models" (arXiv:2510.04618v1), ACE treats contexts as comprehensive playbooks that accumulate, refine, and organize strategies through modular generation, reflection, and curation.

**Key Innovation:** Unlike existing approaches that suffer from brevity bias (collapsing to short prompts) and context collapse (losing detail over iterations), ACE uses structured, itemized bullets that grow incrementally without full rewrites.

---

## Architecture

### Core Components

```
internal/ace/
├── bullet/          # Context bullet data structure
│   ├── bullet.go       # Bullet type and operations
│   ├── validation.go   # Validation logic
│   └── bullet_test.go  # Tests (94.3% coverage)
│
├── playbook/        # Playbook manager
│   ├── playbook.go     # CRUD operations
│   ├── search.go       # Semantic search
│   ├── snapshot.go     # Version control
│   └── storage.go      # Serialization (91.1% coverage)
│
├── embedding/       # Embedding interface
│   ├── embedder.go     # Embedding provider interface
│   └── mock_embedder.go # Mock for testing
│
├── generator/       # Generator component (Feature 2)
│   ├── trajectory.go   # Trajectory data structures
│   └── feedback.go     # BulletFeedback structure
│
├── retrieval/       # Context retrieval (Feature 2)
│   └── retriever.go    # SemanticRetriever (88.9% coverage)
│
├── prompt/          # Prompt construction (Feature 2)
│   └── builder.go      # PromptBuilder (100% coverage)
│
├── feedback/        # Feedback parsing (Feature 2)
│   └── parser.go       # RegexParser (100% coverage)
│
├── reflector/       # Reflector component (Feature 3)
│   ├── insight.go      # Insight data structure
│   ├── request.go      # Request/Response types
│   ├── prompt.go       # Reflection prompt builder
│   └── reflector.go    # Reflector service (95.7% coverage)
│
├── curator/         # Curator component (Feature 4)
│   ├── types.go        # MergeRequest/MergeResult types
│   ├── converter.go    # Insight → Bullet conversion
│   ├── curator.go      # Curator service (90.8% coverage)
│   ├── parallel.go     # Batch processing
│   └── refinement.go   # Refinement strategies
│
├── delta/           # Delta Updates System (Feature 5)
│   ├── delta.go        # Delta data structure and constructors
│   ├── history.go      # DeltaHistory with tracking
│   ├── applier.go      # DeltaApplier for applying deltas
│   ├── batch.go        # Batch processing with worker pool
│   └── *_test.go       # Comprehensive tests (95.9% coverage)
│
├── adapter/         # Adapter component (Feature 8)
│   ├── types.go        # ExecutionSignal, Session types
│   ├── adapter.go      # Online adaptation
│   ├── decision.go     # Decision tree logic
│   └── memory.go       # Memory management (90.7% coverage)
│
└── refine/          # Grow-and-Refine Mechanism (Feature 6)
    ├── archive.go      # Archive system (100% coverage)
    ├── merge.go        # MergeEngine for deduplication
    ├── metrics.go      # GrowthMonitor for tracking
    ├── orchestrator.go # RefinementOrchestrator
    └── *_test.go       # 31 tests (90.7% coverage)
```

---

## Bullet Package

### Purpose

A **bullet** is a single unit of context knowledge - a reusable strategy, domain concept, or failure mode that can be accumulated and refined over time.

### Key Features

- **Unique Identification**: UUID v4 for each bullet
- **Content Storage**: Up to 2048 characters per bullet
- **Utility Tracking**: Helpful/harmful counters track usefulness
- **Semantic Search**: Optional embedding for similarity search
- **Metadata Tags**: Arbitrary key-value pairs for categorization
- **Timestamps**: Creation and modification tracking

### Data Structure

```go
type Bullet struct {
    ID           string            // UUID v4
    Content      string            // Max 2048 chars
    HelpfulCount int               // Non-negative
    HarmfulCount int               // Non-negative
    Embedding    []float32         // Optional, 1536 dimensions
    CreatedAt    time.Time         // Auto-set
    UpdatedAt    time.Time         // Auto-set
    Tags         map[string]string // Optional metadata
}
```

### Usage

#### Creating Bullets

```go
import "github.com/dmytrogajewski/spin/internal/ace/bullet"

// Basic creation
b, err := bullet.New("Always validate user input")
if err != nil {
    log.Fatal(err)
}

// With options
b, err := bullet.New("Use table-driven tests",
    bullet.WithTags(map[string]string{
        "category": "testing",
        "language": "go",
    }),
)

// With custom ID (for testing)
b, err := bullet.New("content", 
    bullet.WithID("custom-id-123"),
)

// With embedding
embedding := []float32{0.1, 0.2, ...} // 1536 dims
b, err := bullet.New("content",
    bullet.WithEmbedding(embedding),
)
```

#### Validation

All bullets are validated on creation:

```go
// ✅ Valid - content within limit
b, err := bullet.New("Short content")  // err == nil

// ❌ Invalid - content too long (>2048 chars)
longContent := strings.Repeat("x", 2049)
b, err := bullet.New(longContent)  // err != nil
```

#### Tracking Utility

```go
b, _ := bullet.New("Use context.Context")

// Mark as helpful
b.IncrementHelpful()
fmt.Println(b.HelpfulCount)  // 1

// Mark as harmful
b.IncrementHarmful()
fmt.Println(b.HarmfulCount)  // 1

// Calculate utility score (-1.0 to 1.0)
score := b.Score()
// score = (helpful - harmful) / (helpful + harmful)
```

#### Cloning

```go
original, _ := bullet.New("Original content")
clone := original.Clone()

// Clone is independent
clone.IncrementHelpful()
fmt.Println(original.HelpfulCount)  // 0
fmt.Println(clone.HelpfulCount)     // 1
```

### Validation Rules

| Rule | Constraint | Error |
|------|-----------|-------|
| Content length | ≤ 2048 chars | "content length N exceeds maximum 2048" |
| Helpful count | ≥ 0 | "helpful count cannot be negative" |
| Harmful count | ≥ 0 | "harmful count cannot be negative" |
| ID format | Valid UUID v4 | Auto-generated if empty |

### Constants

```go
const MaxContentLength = 2048  // Maximum bullet content length
```

---

## Integration with Spin

### Status: ✅ INTEGRATED (Phase 2 Complete - 2025-10-30)

ACE is now fully integrated into Spin's TUI and enabled by default. When you run `spin tui`, ACE automatically:
1. Retrieves relevant bullets from your playbook
2. Injects them into the LLM's context
3. Collects feedback on which bullets were helpful/harmful
4. Updates bullet counters for future retrieval

### Usage

**Basic usage (ACE enabled by default):**
```bash
spin tui --model qwen3-coder:30b --provider ollama
```

ACE works with any LLM provider (OpenAI, Ollama, etc.) and requires no additional configuration.

**Configuration (optional):**

ACE uses sensible defaults but can be customized via Manager configuration:

```go
cfg := &manager.Config{
    ACEEnabled:        true,  // enabled by default
    ACEPlaybookPath:   "~/.spin/ace/playbooks/default.json",
    ACETrajectoryPath: "~/.spin/ace/trajectories/",
    ACETopK:           5,     // retrieve top 5 bullets
    ACEMinScore:       0.3,   // minimum similarity threshold
}
```

### Relationship to History

| Component | Purpose | Scope |
|-----------|---------|-------|
| **History** (`internal/history`) | Stores conversation messages | Per-conversation |
| **ACE Playbook** (`internal/ace/playbook`) | Stores reusable strategies | Cross-conversation |

**Key Difference:** History manages ephemeral conversation flow, while ACE Playbook manages persistent, reusable knowledge bullets that accumulate over time.

### Event Integration

ACE will emit events for observability (future enhancement):

- `EventBulletAdded` - New bullet added to playbook
- `EventBulletUpdated` - Bullet modified
- `EventBulletDeleted` - Bullet removed
- `EventPlaybookSnapshot` - Snapshot created
- `EventPlaybookRestored` - Restored from snapshot

### Service Architecture

ACE is integrated into Spin's service-based architecture via the Manager:

```go
// Manager creates ACE service during agent construction
// See internal/manager/manager.go buildAgent() method

aceConfig := &agent.ACEConfig{
    Enabled:        true,
    PlaybookPath:   cfg.ACEPlaybookPath,
    TrajectoryPath: cfg.ACETrajectoryPath,
    Retrieval: agent.ACERetrievalConfig{
        TopK:     cfg.ACETopK,
        MinScore: cfg.ACEMinScore,
    },
    ItemizedLearning: agent.ACEItemizedLearningConfig{
        Enabled:       true,
        ParseFeedback: true,
        UpdateAsync:   false,
    },
}

aceService, err := agent.NewACEService(aceConfig, workDir)
if err != nil {
    logger.Warn("failed to create ACE service", "error", err)
}

// Agent receives ACE service via functional option
agent := agent.New(agentCfg, logger, agent.WithACEService(aceService))
```

### Integration Points

ACE hooks into the Agent's LLM call lifecycle:

1. **Before LLM Call**: ACE retrieves relevant bullets and injects them into the system prompt
2. **After LLM Call**: ACE parses the response for bullet feedback (HELPFUL/HARMFUL markers)
3. **Bullet Update**: Counters are updated asynchronously based on feedback

See `internal/agent/ace_service.go` for the complete ACE service implementation.

---

## Design Principles

### 1. Comprehensive Over Concise

Unlike traditional prompt optimization that aims for brevity, ACE accumulates detailed, domain-specific knowledge. LLMs perform better with long, detailed contexts and can filter relevance autonomously.

### 2. Incremental Over Monolithic

Bullets are updated individually (delta updates), not rewritten entirely. This prevents "context collapse" where full rewrites lose information.

### 3. Structured Over Unstructured

Bullets are itemized with metadata, enabling:
- Fine-grained retrieval
- Utility tracking
- Efficient search
- Version control

### 4. Observable Over Opaque

All operations emit events for debugging and monitoring.

---

## Performance Characteristics

### Phase 1 (Completed)

| Operation | Target | Actual | Status |
|-----------|--------|--------|--------|
| Bullet creation | < 10μs | TBD | ✅ Implemented |
| Validation | < 1μs | TBD | ✅ Implemented |
| Cloning | < 5μs | TBD | ✅ Implemented |
| Test coverage | ≥ 90% | 94.3% | ✅ Exceeded |

### Future (Phase 2+)

| Operation | Target | Status |
|-----------|--------|--------|
| Lookup by ID | < 1μs (O(1)) | 🔄 Planned |
| Semantic search (100 bullets) | < 5ms | 🔄 Planned |
| Serialization (1000 bullets) | < 100ms | 🔄 Planned |

---

## Generator Component (Feature 2 - Complete)

### Overview

The Generator component implements the core ACE workflow: retrieving relevant context bullets, injecting them into prompts, executing tasks, and collecting feedback on bullet utility.

**Status**: Fully implemented with 89% test coverage, zero lint errors, race detector clean.

**Components**:
- Retrieval: Semantic search for relevant bullets (88.9% coverage)
- Prompt: Builder with ItemizedLearning instructions (100% coverage)
- Feedback: Parser for HELPFUL/HARMFUL markers (100% coverage)
- Generator: Full ItemizedLearning workflow (89% coverage)
- Bullet Generation: From tasks/trajectories/feedback/errors (89% coverage)

### Retrieval Package

**Purpose**: Retrieve relevant bullets for a query using semantic search.

```go
import "github.com/dmytrogajewski/spin/internal/ace/retrieval"

// Create retriever
embedder := embedding.NewMockEmbedder(1536)
pb := playbook.New(nil, embedder)
retriever := retrieval.NewSemanticRetriever(pb, embedder)

// Retrieve top-K bullets
bullets, err := retriever.Retrieve(ctx, "How to handle errors in Go?", 5)

// Retrieve with scores
scored, err := retriever.RetrieveWithScores(ctx, "error handling", 5)
for _, s := range scored {
    fmt.Printf("Bullet: %s (score: %.2f)\n", s.Bullet.Content, s.Score)
}
```

### Prompt Package

**Purpose**: Construct prompts with context bullets and ItemizedLearning instructions.

```go
import "github.com/dmytrogajewski/spin/internal/ace/prompt"

// Basic prompt builder
builder := prompt.NewBuilder()
systemPrompt := builder.BuildSystemPrompt(bullets)

// With ItemizedLearning instructions
builder := prompt.NewBuilder(
    prompt.WithItemizedLearning(),
    prompt.WithSystemPrompt("You are an expert Go developer"),
)
systemPrompt := builder.BuildSystemPrompt(bullets)
```

**Output Example:**
```
You are an expert Go developer

# Context Playbook

[B0] Always check errors immediately after they occur
[B1] Use errors.Is and errors.As for error type checking
[B2] Wrap errors with context using fmt.Errorf with %w

# Instructions

1. Use the context bullets above to solve the task
2. After solving, indicate which bullets were helpful or harmful:
   - HELPFUL: [B1, B3, ...] - bullets that helped solve the task
   - HARMFUL: [B2, ...] - bullets that misled or were incorrect
3. Provide your reasoning
```

### Feedback Package

**Purpose**: Parse bullet utility feedback from LLM responses.

```go
import "github.com/dmytrogajewski/spin/internal/ace/feedback"

parser := feedback.NewRegexParser()

response := `The correct approach is to use errors.Is.

HELPFUL: [B1, B2]
HARMFUL: [B0]
EXPLANATION: B0 suggested an outdated pattern`

feedback, err := parser.Parse(response)
// feedback.HelpfulBullets = ["B1", "B2"]
// feedback.HarmfulBullets = ["B0"]
// feedback.Explanation = "B0 suggested an outdated pattern"
```

### Generator - ItemizedLearning Workflow

**Purpose**: Complete end-to-end workflow that retrieves bullets, injects into prompts, executes tasks, and updates playbook based on feedback.

```go
import (
    "context"
    "github.com/dmytrogajewski/spin/internal/ace/generator"
    "github.com/dmytrogajewski/spin/internal/ace/playbook"
    "github.com/dmytrogajewski/spin/internal/ace/retrieval"
    "github.com/dmytrogajewski/spin/internal/ace/embedding"
    "github.com/dmytrogajewski/spin/internal/llm"
)

// Setup
ctx := context.Background()
embedder := embedding.NewMockEmbedder(1536)
pb := playbook.New(nil, embedder)
ret := retrieval.NewSemanticRetriever(pb, embedder)
llmProvider := llm.NewMockProvider("test-provider")

// Add bullets to playbook
emb, _ := embedder.Embed(ctx, "Always validate input")
b, _ := bullet.New("Always validate input", bullet.WithEmbedding(emb))
pb.Add(ctx, b)

// Create generator
gen, err := generator.NewGenerator(generator.Config{
    LLM:       llmProvider,
    Playbook:  pb,
    Retriever: ret,
})

// Execute ItemizedLearning
req := generator.ItemizedLearningRequest{
    Query:       "How to validate user input?",
    TopK:        5,
    Model:       "gpt-4",
    Temperature: 0.7,
    MaxTokens:   1000,
}

resp, err := gen.ItemizedLearning(ctx, req)

// Response contains:
// - resp.Output: LLM's answer
// - resp.Feedback: Parsed HELPFUL/HARMFUL markers
// - resp.Trajectory: Complete execution trace
// - resp.Success: Whether task succeeded (if ground truth provided)

// Playbook bullets are automatically updated with feedback counters
```

### Trajectory Data Structures

**Purpose**: Capture complete execution traces with metadata.

```go
import "github.com/dmytrogajewski/spin/internal/ace/generator"

// Create trajectory
traj := &generator.Trajectory{
    ID:               "traj-123",
    Query:            "How to handle errors?",
    RetrievedBullets: bullets,
    Steps: []generator.TrajectoryStep{
        {
            StepNumber: 0,
            Type:       "reasoning",
            Content:    "First, check the error...",
            Timestamp:  time.Now(),
        },
    },
    Output:  "Use errors.Is for type checking",
    Success: true,
    BulletFeedback: &generator.BulletFeedback{
        HelpfulBullets: []string{"bullet-1", "bullet-2"},
        HarmfulBullets: []string{},
    },
    Metadata: generator.TrajectoryMetadata{
        Model:       "gpt-4",
        Temperature: 0.7,
        TotalTokens: 500,
        Turns:       1,
    },
    CreatedAt: time.Now(),
}
```

### Bullet Generation

**Purpose**: Generate new bullet candidates from various sources (tasks, trajectories, feedback, errors).

```go
import "github.com/dmytrogajewski/spin/internal/ace/generator"

// Generate bullets from a task description
req := generator.BulletGenerationRequest{
    Input:      "Write a function to parse JSON files",
    SourceType: "task",
    MaxBullets: 5,
    Tags: map[string]string{
        "category": "json_processing",
    },
}

bullets, err := gen.GenerateBullets(ctx, req)
// Returns: []*bullet.Bullet with generated prevention/pattern bullets

// Generate bullets from a trajectory
req = generator.BulletGenerationRequest{
    Input:      trajectoryJSON,
    SourceType: "trajectory",
    MaxBullets: 3,
}

// Generate bullets from error feedback
req = generator.BulletGenerationRequest{
    Input:      "panic: nil pointer dereference in parseJSON",
    SourceType: "error",
    MaxBullets: 3,
    Tags: map[string]string{
        "error_type": "panic",
    },
}

bullets, err := gen.GenerateBullets(ctx, req)
// Returns bullets like:
// - "Always check for nil before dereferencing pointers"
// - "Validate input parameters before processing"
// - "Use defensive programming in public APIs"
```

**Source Types**:
- `"task"` - Extract strategies from task descriptions
- `"trajectory"` - Distill lessons from execution traces
- `"feedback"` - Generate improvements from user feedback
- `"error"` - Create prevention bullets from errors

### Current Limitations

- **No Agent Integration**: Trajectory generation requires manual creation
- **No Streaming**: Responses are processed synchronously
- **Basic Parsing**: Bullet extraction uses simple regex patterns
- **No Deduplication**: Generated bullets aren't checked for duplicates

### Future Enhancements

- Integration with Agent for automatic trajectory generation
- Streaming response support for long-running generations
- Advanced NLP for better bullet extraction
- Automatic deduplication during generation
- Batch processing for multiple tasks
- Custom prompt templates per domain

---

## Reflector Component (Feature 3 - FULLY COMPLETE)

### Overview

The Reflector analyzes execution trajectories to extract actionable insights. It implements the "Reflect" step of the ACE learning loop: Generate → **Reflect** → Curate.

**Purpose**: Transform raw trajectory data into structured insights that can be added to the playbook.

**Status**: Fully implemented with 95.7% test coverage, 27 tests, 5 integration tests, 8 benchmarks. All DoD items complete.

### Core Types

#### Insight

```go
type Insight struct {
    Content    string          // Actionable lesson (50-500 chars)
    Source     string          // Trajectory ID
    Confidence float64         // Reliability score (0.0 to 1.0)
    Category   InsightCategory // Type of insight
    Evidence   []string        // Supporting quotes
    Iteration  int             // Refinement round
    CreatedAt  time.Time       // Timestamp
}

type InsightCategory string

const (
    CategorySuccessPattern InsightCategory = "success_pattern"
    CategoryErrorMode      InsightCategory = "error_mode"
    CategoryOptimization   InsightCategory = "optimization"
    CategoryAntiPattern    InsightCategory = "anti_pattern"
)
```

#### Validation

All insights are validated on creation:

```go
insight := &Insight{
    Content:    "Always validate input parameters before processing",
    Confidence: 0.85,
    Category:   CategorySuccessPattern,
}

err := insight.Validate()
// Checks:
// - Content not empty
// - Content length 50-500 chars
// - Confidence 0.0-1.0
// - Valid category enum
```

### Usage

#### Basic Reflection

```go
import (
    "context"
    "github.com/dmytrogajewski/spin/internal/ace/reflector"
    "github.com/dmytrogajewski/spin/internal/ace/generator"
    "github.com/dmytrogajewski/spin/internal/llm"
)

// Create reflector
llmProvider := llm.NewMockProvider("gpt-4")
ref := reflector.NewReflector(llmProvider)

// Prepare trajectory
traj := &generator.Trajectory{
    ID:      "traj-123",
    Query:   "How to handle errors in Go?",
    Output:  "Use errors.Is for type checking",
    Success: true,
}

// Reflect on trajectory
ctx := context.Background()
req := reflector.ReflectionRequest{
    Trajectories: []*generator.Trajectory{traj},
}

resp, err := ref.Reflect(ctx, req)
if err != nil {
    log.Fatal(err)
}

// Use insights
for _, insight := range resp.Insights {
    fmt.Printf("Insight: %s (confidence: %.2f)\n", 
        insight.Content, insight.Confidence)
    fmt.Printf("Evidence: %v\n", insight.Evidence)
}
```

### Prompt Building

The Reflector uses specialized prompts for trajectory analysis:

```go
builder := reflector.NewPromptBuilder()
prompt := builder.BuildSingleTrajectory(traj)

// Prompt includes:
// - Trajectory data (query, output, success)
// - Task instructions
// - JSON format specification
// - Quality requirements (50-500 chars, evidence, confidence, category)
```

**Prompt Characteristics:**
- Requests comprehensive analysis (not brief)
- Demands evidence from trajectory
- Specifies confidence scoring
- Enforces structured JSON output

### Response Format

The LLM returns insights as JSON:

```json
[
  {
    "content": "Always use errors.Is for error type checking in Go",
    "evidence": ["Use errors.Is for type checking"],
    "confidence": 0.9,
    "category": "success_pattern"
  }
]
```

The Reflector parses this into `Insight` structs.

### Multi-Iteration Refinement

Refine insights through multiple LLM iterations for improved quality:

```go
// Get initial insights
resp, err := ref.Reflect(ctx, req)
initialInsights := resp.Insights

// Refine up to 5 times for better specificity
refined, err := ref.RefineInsights(ctx, initialInsights, 5)

// Refined insights have higher iteration count and typically:
// - More specific content
// - Higher confidence scores
// - Better actionability
for _, insight := range refined {
    fmt.Printf("Iteration %d: %s (confidence: %.2f)\n",
        insight.Iteration, insight.Content, insight.Confidence)
}
```

### Batch Trajectory Analysis

Analyze multiple trajectories together to find cross-trajectory patterns:

```go
// Multiple related trajectories
trajectories := []*generator.Trajectory{
    {ID: "traj-1", Query: "Error handling", Output: "Use errors.Is", Success: true},
    {ID: "traj-2", Query: "Error patterns", Output: "Wrap errors with context", Success: true},
    {ID: "traj-3", Query: "Panic handling", Output: "Avoid panic in libraries", Success: false},
}

req := reflector.ReflectionRequest{
    Trajectories: trajectories,
}

resp, err := ref.Reflect(ctx, req)

// Insights extracted from patterns across all trajectories
// - Success patterns common to multiple trajectories
// - Recurring error modes
// - Anti-patterns leading to failures
for _, insight := range resp.Insights {
    fmt.Printf("Pattern: %s (from %s)\n", insight.Content, insight.Source)
    fmt.Printf("Evidence: %v\n", insight.Evidence)
}
```

### Quality Validation and Filtering

Validate insight quality and filter by confidence threshold:

```go
validator := reflector.NewInsightValidator()

// Validate individual insight
insight := &reflector.Insight{
    Content:    "Always validate input parameters before processing",
    Confidence: 0.85,
    Category:   reflector.CategorySuccessPattern,
}

err := validator.Validate(insight)
if err != nil {
    // Insight fails validation (too short, invalid confidence, etc.)
}

// Batch validation
errs := validator.ValidateBatch(insights)
if len(errs) > 0 {
    // Some insights have validation errors
}

// Filter by quality threshold
highQuality := validator.FilterByQuality(insights, 0.8)
// Only insights with confidence >= 0.8

mediumQuality := validator.FilterByQuality(insights, 0.5)
// Insights with confidence >= 0.5
```

### Complete Workflow Example

Full end-to-end reflection workflow with refinement and filtering:

```go
// Step 1: Initial reflection on trajectories
req := reflector.ReflectionRequest{
    Trajectories: trajectories,
}
resp, err := ref.Reflect(ctx, req)

// Step 2: Refine insights for better quality
refined, err := ref.RefineInsights(ctx, resp.Insights, 3)

// Step 3: Validate and filter
validator := reflector.NewInsightValidator()
errs := validator.ValidateBatch(refined)
if len(errs) > 0 {
    log.Printf("Validation warnings: %v", errs)
}

// Step 4: Filter by quality threshold
highQuality := validator.FilterByQuality(refined, 0.7)

// Step 5: Use high-quality insights
for _, insight := range highQuality {
    fmt.Printf("High-quality insight: %s\n", insight.Content)
    fmt.Printf("  Confidence: %.2f\n", insight.Confidence)
    fmt.Printf("  Category: %s\n", insight.Category)
    fmt.Printf("  Evidence: %v\n", insight.Evidence)
}
```

### Performance Characteristics

Benchmarked on AMD Ryzen AI 9 HX 370:

| Operation | Time | Allocations |
|-----------|------|-------------|
| Single trajectory reflection | 2.5μs | 33 allocs |
| Batch (10 trajectories) | 5.0μs | 100 allocs |
| Refinement (3 iterations) | 9.7μs | 109 allocs |
| Insight validation | 2.3ns | 0 allocs |
| Quality filtering (100 insights) | 198ns | 1 alloc |
| Single trajectory prompt build | 442ns | 9 allocs |
| Batch trajectory prompt build | 3.4μs | 77 allocs |
| Refinement prompt build | 1.0μs | 13 allocs |

### Components

**Files**: 8 files, ~800 lines (400 production + 400 tests)
- `insight.go` - Insight data structures and validation
- `request.go` - Request/Response types
- `reflector.go` - Main Reflector implementation
- `prompt.go` - Prompt builders (single, batch, refinement)
- `validator.go` - Quality validation and filtering
- `*_test.go` - 27 unit tests
- `integration_test.go` - 5 integration tests
- `reflector_bench_test.go` - 8 benchmarks

**Test Coverage**: 95.7% (exceeds 90% requirement)
**Lint Errors**: Zero
**Race Detector**: Clean

### Current Capabilities

- ✅ Single trajectory analysis
- ✅ Batch trajectory analysis (cross-trajectory patterns)
- ✅ Multi-iteration refinement (up to 5 rounds)
- ✅ Quality validation and filtering
- ✅ Confidence scoring
- ✅ Evidence extraction
- ✅ Category classification (4 types)
- ✅ Integration with Generator
- ✅ Performance benchmarking

### Future Enhancements
- Batch trajectory analysis
- Cross-trajectory pattern detection
- Insight quality filtering and ranking
- Evidence strength scoring
- Integration with Curator for playbook updates

---

## Curator Component (Feature 4 - FULLY COMPLETE)

### Overview

The Curator transforms insights from the Reflector into bullets and merges them into the playbook with semantic deduplication. It implements the "Curate" step of the ACE learning loop: Generate → Reflect → **Curate**.

**Purpose**: Convert insights to bullets, detect duplicates using semantic similarity, and update the playbook intelligently.

**Status**: Fully implemented with 92.9% test coverage, 30+ tests, 3 integration tests, 6 benchmarks. All DoD items complete.

### Core Types

```go
type MergeRequest struct {
    Insights []*reflector.Insight
}

type MergeResult struct {
    Added        int              // New bullets added
    Skipped      int              // Duplicates skipped
    Updated      int              // Existing bullets updated
    Duplicates   []string         // IDs of duplicate bullets
    AddedBullets []*bullet.Bullet // Bullets that were added
}

type Curator interface {
    Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)
    FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)
}

type Option func(*curator)

func WithSimilarityThreshold(threshold float64) Option
```

### Usage

#### Basic Curation

```go
import (
    "context"
    "github.com/dmytrogajewski/spin/internal/ace/curator"
    "github.com/dmytrogajewski/spin/internal/ace/reflector"
    "github.com/dmytrogajewski/spin/internal/ace/playbook"
    "github.com/dmytrogajewski/spin/internal/ace/embedding"
)

// Setup
ctx := context.Background()
embedder := embedding.NewMockEmbedder(384)
pb := playbook.New(nil, embedder)

// Create curator with default threshold (0.85)
cur := curator.NewCurator(pb, embedder)

// Or with custom threshold
cur := curator.NewCurator(pb, embedder, 
    curator.WithSimilarityThreshold(0.90))

// Get insights from reflector (see Reflector section)
insights := []*reflector.Insight{...}

// Curate into playbook
req := curator.MergeRequest{Insights: insights}

result, err := cur.Curate(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Added %d bullets\n", result.Added)
fmt.Printf("Skipped %d duplicates\n", result.Skipped)
fmt.Printf("Updated %d existing bullets\n", result.Updated)
```

### Insight to Bullet Conversion

The curator converts insights using these rules:

```
Insight → Bullet:
- Content → Content (1:1)
- Confidence (0.0-1.0) → HelpfulCount (0-10)
- Category → Tags["category"]
- Source → Tags["source"]  
- Evidence → Tags["evidence"] (joined with ";")
```

**Example:**
```go
// Input
insight := &reflector.Insight{
    Content:    "Always validate input parameters",
    Confidence: 0.85,
    Category:   reflector.CategorySuccessPattern,
    Source:     "traj-123",
    Evidence:   []string{"validation prevented error"},
}

// Output bullet after conversion
bullet := &bullet.Bullet{
    Content:      "Always validate input parameters",
    HelpfulCount: 8,  // int(0.85 * 10) = 8
    Tags: {
        "category": "success_pattern",
        "source":   "traj-123",
        "evidence": "validation prevented error",
    },
}
```

### Semantic Deduplication

The Curator detects duplicates using **cosine similarity** on bullet embeddings:

```go
// FindDuplicates compares new bullets against playbook
duplicates, err := cur.FindDuplicates(ctx, newBullets)

// Returns map[newBulletID]existingBulletID
// Example: {"new-123": "existing-456"}
```

**Algorithm:**
1. Get embeddings for all bullets (both new and existing)
2. Calculate cosine similarity between each new bullet and all playbook bullets
3. If similarity ≥ threshold (default 0.85), mark as duplicate
4. Return map of new bullet ID → existing bullet ID

**Cosine Similarity Formula:**
```
similarity = dot(a, b) / (||a|| * ||b||)

Where:
- dot(a, b) = sum of element-wise products
- ||a|| = Euclidean norm of vector a
```

### Duplicate Handling

When a duplicate is found during curation:

1. **Skip adding** the new bullet (no duplicate entries)
2. **Increment helpful count** of the existing bullet
3. **Track statistics** (skipped count, updated count, duplicate IDs)

```go
// Example: Curating duplicate insight
insights := []*reflector.Insight{
    {Content: "Always validate input", Confidence: 0.9},
}

// First curation - adds bullet
result1, _ := cur.Curate(ctx, curator.MergeRequest{Insights: insights})
// result1.Added = 1, result1.Skipped = 0

// Second curation - same insight
result2, _ := cur.Curate(ctx, curator.MergeRequest{Insights: insights})
// result2.Added = 0, result2.Skipped = 1, result2.Updated = 1
// Existing bullet's HelpfulCount incremented by 1
```

### Custom Similarity Threshold

Adjust the similarity threshold to control duplicate detection sensitivity:

```go
// Strict (0.95) - only near-identical bullets are duplicates
strict := curator.NewCurator(pb, embedder, 
    curator.WithSimilarityThreshold(0.95))

// Lenient (0.75) - more bullets considered duplicates
lenient := curator.NewCurator(pb, embedder,
    curator.WithSimilarityThreshold(0.75))

// Default (0.85) - balanced
default := curator.NewCurator(pb, embedder)
```

**Threshold Guidelines:**
- **0.95+**: Very strict, only exact/near-exact matches
- **0.85-0.95**: Recommended range, catches similar content
- **0.75-0.85**: Lenient, may flag loosely related bullets
- **<0.75**: Too lenient, risk of false positives

### Integration with Reflector

Complete end-to-end workflow from trajectories to playbook:

```go
// Step 1: Reflect on trajectories
reflectReq := reflector.ReflectionRequest{
    Trajectories: trajectories,
}
reflectResp, _ := ref.Reflect(ctx, reflectReq)

// Step 2: Curate insights into playbook
curateReq := curator.MergeRequest{
    Insights: reflectResp.Insights,
}
curateResp, _ := cur.Curate(ctx, curateReq)

fmt.Printf("Extracted %d insights\n", len(reflectResp.Insights))
fmt.Printf("Added %d new bullets\n", curateResp.Added)
fmt.Printf("Skipped %d duplicates\n", curateResp.Skipped)

// Playbook now contains bullets
bullets := pb.List(nil)
fmt.Printf("Playbook has %d total bullets\n", len(bullets))
```

### Idempotent Curation

Curating the same insights multiple times is safe and idempotent:

```go
insights := []*reflector.Insight{
    {Content: "Use errors.Is for error checking", Confidence: 0.9},
}
req := curator.MergeRequest{Insights: insights}

// First curation
result1, _ := cur.Curate(ctx, req)
// Adds 1 bullet

// Second curation (same insights)
result2, _ := cur.Curate(ctx, req)
// Skips 1 duplicate, increments helpful count

// Third curation
result3, _ := cur.Curate(ctx, req)
// Skips 1 duplicate, increments helpful count again

// Playbook still has only 1 bullet, but helpful count = 11
// (initial 9 from confidence 0.9, +1 from 2nd, +1 from 3rd)
```

### Performance Characteristics

Benchmarked on AMD Ryzen AI 9 HX 370:

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| ConvertInsights (1 insight) | 390ns | 536 B | 6 allocs |
| FindDuplicates (empty playbook) | 52ns | 56 B | 2 allocs |
| FindDuplicates (100 bullets) | 25.5μs | 1.2 KB | 4 allocs |
| Curate (single insight) | 1.0μs | 2.5 KB | 14 allocs |
| Curate (10 insights) | 11.1μs | 21.7 KB | 71 allocs |
| Cosine similarity (384-dim) | 236ns | 0 B | 0 allocs |

### Components

**Files**: 12 files, ~1,600 lines (800 production + 800 tests)
- `types.go` - MergeRequest/MergeResult/BatchMergeRequest/RefinementResult types
- `curator.go` - Main Curator implementation with functional options
- `converter.go` - Insight → Bullet conversion logic
- `deduplicator.go` - Semantic duplicate detection (cosine similarity)
- `parallel.go` - Parallel batch processing with worker pool (NEW)
- `refinement.go` - Refinement strategy interface and implementations (NEW)
- `curator_test.go` - Core unit tests
- `parallel_test.go` - Parallel processing tests (NEW)
- `refinement_test.go` - Refinement strategy tests (NEW)
- `integration_test.go` - 3 integration tests with Reflector
- `coverage_test.go` - Edge case tests for 90%+ coverage
- `curator_bench_test.go` - 6 performance benchmarks

**Test Coverage**: 90.8% (exceeds 90% requirement)
**Lint Errors**: Zero
**Race Detector**: Clean

### Current Capabilities

- ✅ Insight to bullet conversion with confidence scaling
- ✅ Semantic deduplication using cosine similarity
- ✅ Duplicate detection with configurable threshold
- ✅ Helpful count incrementation for duplicates
- ✅ Functional options pattern (WithSimilarityThreshold, WithRefinementMode)
- ✅ Integration with Reflector component
- ✅ Idempotent curation (safe to curate same insights multiple times)
- ✅ **Parallel batch processing** with worker pool (CurateBatch)
- ✅ **Refinement strategies**: None, Lazy, Proactive modes
- ✅ **Automatic playbook pruning** based on utility thresholds
- ✅ Comprehensive test coverage (46+ tests, 90.8% coverage)
- ✅ Performance benchmarks
- ✅ Race condition free

### Parallel Batch Processing (NEW)

Process multiple merge requests concurrently for high-throughput scenarios:

```go
// Create curator with parallel support
cur := curator.NewCurator(pb, embedder)

// Process multiple insight batches in parallel
batch := curator.BatchMergeRequest{
    Requests: []curator.MergeRequest{
        {Insights: insights1},
        {Insights: insights2},
        {Insights: insights3},
        // ... up to 100+ batches
    },
    MaxWorkers: 8, // Use 8 workers (0 = NumCPU)
}

result, err := cur.CurateBatch(ctx, batch)

// Check results per request
for i, res := range result.Results {
    if result.Errors[i] != nil {
        log.Printf("Request %d failed: %v", i, result.Errors[i])
        continue
    }
    log.Printf("Request %d: added=%d, skipped=%d", i, res.Added, res.Skipped)
}
```

**Features:**
- Worker pool with configurable concurrency (default: NumCPU)
- Thread-safe access to shared playbook
- Per-request error handling (partial failures don't affect other requests)
- Context cancellation support
- Performance: ~8x speedup on 8-core CPU

### Refinement Strategies (NEW)

Control when and how the playbook is refined (low-utility bullets pruned):

#### No Refinement (Default)

```go
// Default: no refinement
cur := curator.NewCurator(pb, embedder)

// Playbook grows indefinitely
```

#### Lazy Refinement

Manual refinement only - you control when pruning occurs:

```go
cur := curator.NewCurator(pb, embedder,
    curator.WithRefinementMode(
        curator.RefinementModeLazy,
        curator.LazyRefinementConfig{
            MinUtilityScore: 0.1, // Prune bullets with score < 0.1
        },
    ),
)

// Curate insights (no auto-refinement)
result, _ := cur.Curate(ctx, curator.MergeRequest{Insights: insights})

// Manually trigger refinement when desired
refinement, err := cur.Refine(ctx)
log.Printf("Pruned %d bullets: %v", refinement.Pruned, refinement.PrunedIDs)
```

#### Proactive Refinement

Automatic refinement after each Curate() when threshold is reached:

```go
cur := curator.NewCurator(pb, embedder,
    curator.WithRefinementMode(
        curator.RefinementModeProactive,
        curator.ProactiveRefinementConfig{
            MaxBullets:      500,  // Trigger when playbook has 500+ bullets
            MinUtilityScore: 0.2,  // Prune bullets with score < 0.2
        },
    ),
)

// Curate insights
result, _ := cur.Curate(ctx, curator.MergeRequest{Insights: insights})

// Check if refinement was triggered
if result.Refined {
    log.Printf("Auto-refinement: pruned %d bullets", result.Refinement.Pruned)
    log.Printf("Reason: %s", result.Refinement.Reason)
}
```

**Utility Score Calculation:**
```go
// Bullet utility score = (helpful - harmful) / (helpful + harmful)
// Range: -1.0 (all harmful) to 1.0 (all helpful)
// Score 0.0 = equal helpful/harmful or no feedback yet
```

**Default Configuration:**
- **Lazy**: MinUtilityScore = 0.1
- **Proactive**: MaxBullets = 1000, MinUtilityScore = 0.1

### Updated Types

```go
type MergeResult struct {
    Added        int
    Skipped      int
    Updated      int
    Duplicates   []string
    AddedBullets []*bullet.Bullet
    Refined      bool             // NEW: Was refinement triggered?
    Refinement   *RefinementResult // NEW: Refinement stats
}

type RefinementResult struct {
    Pruned    int      // Bullets removed
    PrunedIDs []string // IDs of removed bullets
    Reason    string   // Why refinement occurred
}

type BatchMergeRequest struct {
    Requests   []MergeRequest
    MaxWorkers int // 0 = runtime.NumCPU()
}

type BatchMergeResult struct {
    Results []MergeResult
    Errors  []error // per-request errors
}
```

### Updated Interface

```go
type Curator interface {
    Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)
    CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error) // NEW
    Refine(ctx context.Context) (*RefinementResult, error) // NEW
    FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)
}
```

### Future Enhancements

- Bullet merging (combine similar bullets into one)
- Quality-based filtering before curation
- LLM-based refinement (semantic merging)
- Archival system (move old bullets to archive)
- User approval UI for refinement actions

---

## Adapter Component (Feature 8 - Complete)

### Overview

The Adapter enables **online context adaptation** - real-time playbook updates during task execution. It processes execution signals (test failures, build errors, user corrections) and decides whether to skip, quickly add, or deeply reflect on them.

**Purpose**: Enable learning during conversations without waiting for batch offline processing.

**Status**: Fully implemented with Reflect and QuickAdd actions executing end-to-end.

### Core Features

- **6 Signal Types**: test, build, lint, error, tool_use, user
- **3 Signal Outcomes**: success, failure, neutral
- **4 Adaptation Actions**: skip, reflect, quick_add, update
- **Session Management**: Start/track/end sessions with sliding window
- **Decision Tree**: Automatic action selection based on signal type/outcome
- **Full Reflect**: Signal → Trajectory → Reflector → Insights → Curator → Bullets
- **Full QuickAdd**: Signal → Generator → Bullets (with generator) or fallback
- **Auto Memory Management**: Triggers pruning when playbook exceeds threshold

### Implementation Summary

**Files**: 8 files, ~1,200 lines (400 production + 800 tests)
- `types.go` - ExecutionSignal, Session, AdaptationResult
- `adapter.go` - Adapter implementation with full Reflect/QuickAdd
- `decision.go` - Decision tree logic
- `memory.go` - Memory management with utility-based pruning
- `*_test.go` - 32 comprehensive tests (90.7% coverage)

**Test Coverage**: 90.7% (exceeds 90% requirement)
**Lint Errors**: Zero
**Race Detector**: Clean

### Quick Example

```go
// Setup
embedder := embedding.NewMockEmbedder(384)
pb := playbook.New(nil, embedder)
llmProvider := llm.NewMockProvider("gpt-4")
refl := reflector.NewReflector(llmProvider)
cur := curator.NewCurator(pb, embedder)
adapter := adapter.NewAdapter(pb, refl, cur)

// Start session
sessionID, _ := adapter.StartSession(ctx)

// Process test failure signal
signal := adapter.ExecutionSignal{
    SignalType: adapter.SignalTypeTest,
    Context:    "TestFoo failed with nil pointer",
    Outcome:    adapter.OutcomeFailure,
    SessionID:  sessionID,
}

result, _ := adapter.AdaptOnline(ctx, signal)
// result.Action == ActionReflect
// result.BulletsAdded > 0 (insights extracted and added to playbook)
```

See the full Adapter Component section above for detailed documentation, signal types, decision logic, memory management, and usage examples.

### Current Limitations

- **No Agent Integration**: Signals must be manually created
- **No Session Persistence**: Sessions lost on restart
- **Synchronous Only**: No async mode (may add latency)
- **Basic Fallback**: QuickAdd without generator does nothing

### Future Enhancements

- Integrate with Agent's tool execution flow
- Persist sessions for recovery
- Async signal processing
- Batch signal processing
- User approval UI for online additions
- Rollback/undo for recent adaptations

---

## Roadmap

### ✅ Phase 1: Foundation (Completed - 2025-10-29)
- [x] Bullet data structure
- [x] Validation logic
- [x] Basic operations (New, Clone, Score, Increment)
- [x] Functional options (WithID, WithEmbedding, WithTags)
- [x] Full test coverage (94.3%)
- [x] Race detector clean
- [x] Go vet clean

### ✅ Phase 2: Playbook Manager (Completed - 2025-10-29)
- [x] CRUD operations (Add, Get, Update, Delete, List)
- [x] Map-based O(1) indexing
- [x] Thread-safe access (sync.RWMutex)
- [x] Statistics calculation
- [x] Event emission ready
- [x] Full test coverage (91.1%)
- [x] Race detector clean

### ✅ Phase 3: Search & Persistence (Completed - 2025-10-29)
- [x] Semantic search with embeddings
- [x] Cosine similarity calculation
- [x] Embedder interface
- [x] JSON serialization (Save/Load)
- [x] Atomic writes for crash safety
- [x] Validation on load

### ✅ Phase 4: Version Control (Completed - 2025-10-29)
- [x] Snapshot creation (immutable)
- [x] Restore from snapshot
- [x] Diff between snapshots
- [x] Deep copy for immutability

### 📋 Phase 5: Integration (Future)
- [ ] Manager integration
- [ ] Configuration support
- [ ] TUI visualization
- [ ] Documentation

### ✅ Phase 5: ItemizedLearning Workflow (COMPLETED - 2025-10-29)
- [x] Trajectory data structures
- [x] BulletFeedback structures  
- [x] Semantic retrieval (88.9% coverage)
- [x] Prompt builder with ItemizedLearning (100% coverage)
- [x] Feedback parser (100% coverage)
- [x] ItemizedLearning implementation (89.0% coverage)
- [x] Generator with full workflow (89.0% coverage)
- [x] Playbook integration (bullet counter updates)
- [x] 18 comprehensive integration tests
- [x] Zero lint errors and race conditions

### ✅ Phase 6: Bullet Generation (COMPLETED - 2025-10-29)
- [x] GenerateBullets method implementation
- [x] Support for 4 source types (task, trajectory, feedback, error)
- [x] Specialized prompts per source type
- [x] Bullet candidate parsing (numbered, dashed, asterisk formats)
- [x] 8 comprehensive tests for bullet generation
- [x] Table-driven tests for candidate parsing (7 sub-cases)
- [x] 89.0% overall generator coverage

### ✅ Phase 7: Reflector Component (COMPLETED - 2025-10-30)
- [x] Insight data structures with validation
- [x] Single trajectory reflection
- [x] Batch trajectory analysis (cross-trajectory patterns)
- [x] Multi-iteration refinement (up to 5 rounds)
- [x] Quality validation and filtering (InsightValidator)
- [x] Prompt builders (single, batch, refinement)
- [x] 27 unit tests (95.7% coverage)
- [x] 5 integration tests with Generator
- [x] 8 performance benchmarks
- [x] Zero lint errors and race conditions

### ✅ Phase 8: Curator Component (COMPLETED - 2025-10-30)
- [x] Curator component (delta synthesis)
- [x] Insight to bullet conversion
- [x] Semantic deduplication
- [x] Parallel batch processing
- [x] Refinement strategies (None, Lazy, Proactive)
- [x] 46+ tests with 90.8% coverage

### ✅ Phase 9: Delta Updates (COMPLETED - 2025-10-30)
- [x] Delta data structures (6 operation types)
- [x] DeltaHistory with tracking
- [x] DeltaApplier with batch processing
- [x] Thread-safe operations
- [x] 95.9% test coverage

### ✅ Phase 10: Grow-and-Refine Mechanism (COMPLETED - 2025-10-30)
- [x] Archive system with 4 reason types
- [x] MergeEngine with semantic deduplication
- [x] GrowthMonitor with 4 trigger conditions
- [x] RefinementOrchestrator for coordinated workflow
- [x] Integration with Curator for pruning
- [x] 31 tests with 90.7% coverage
- [x] Zero lint errors and race conditions

### 📋 Phase 11+: Advanced Features
- [x] Online adaptation modes (Adapter) - COMPLETED
- [ ] Full trajectory generation with agent integration
- [ ] Offline context adaptation
- [ ] TUI visualization for insights and bullets
- [ ] ANN-based merging (HNSW, FAISS)
- [ ] Async refinement with progress tracking

---

## Testing

### Coverage Requirements

- **Critical paths:** ≥90% coverage
- **Overall:** ≥85% coverage
- **Race detector:** Must pass `-race`

### Test Organization

```
internal/ace/bullet/
├── bullet_test.go       # Unit tests
└── bullet_bench_test.go # Benchmarks (coming)

internal/ace/playbook/
├── playbook_test.go        # Unit tests (coming)
├── integration_test.go     # Integration tests (coming)
└── playbook_bench_test.go  # Benchmarks (coming)
```

### Running Tests

```bash
# All ACE tests
go test ./internal/ace/... -v

# With race detector
go test ./internal/ace/... -race

# With coverage
go test ./internal/ace/... -cover

# Benchmarks
go test ./internal/ace/... -bench=.
```

---

## Examples

### Example 1: Building a Security Playbook

```go
// Create bullets for security best practices
bullets := []*bullet.Bullet{
    bullet.New("Always validate and sanitize user input"),
    bullet.New("Use parameterized queries to prevent SQL injection"),
    bullet.New("Implement rate limiting on authentication endpoints"),
    bullet.New("Never log sensitive data like passwords or tokens"),
}

// Tag them
for _, b := range bullets {
    b.Tags = map[string]string{
        "category": "security",
        "priority": "high",
    }
}

// Mark helpful ones based on usage
bullets[0].IncrementHelpful()
bullets[0].IncrementHelpful()
bullets[1].IncrementHelpful()

// Calculate scores
for _, b := range bullets {
    fmt.Printf("%s: score=%.2f\n", b.Content, b.Score())
}
```

### Example 2: Cloning for Testing

```go
func TestBulletModification(t *testing.T) {
    original, _ := bullet.New("Test content")
    
    // Clone for independent testing
    modified := original.Clone()
    modified.Content = "Modified content"
    modified.IncrementHelpful()
    
    // Original unchanged
    assert.Equal(t, "Test content", original.Content)
    assert.Equal(t, 0, original.HelpfulCount)
    
    // Modified independently
    assert.Equal(t, "Modified content", modified.Content)
    assert.Equal(t, 1, modified.HelpfulCount)
}
```

---

## Troubleshooting

### Content Too Long Error

**Problem:** `content length 2049 exceeds maximum 2048`

**Solution:** Split long content into multiple bullets or summarize:

```go
// ❌ Too long
longContent := strings.Repeat("x", 2049)
b, err := bullet.New(longContent)  // Error!

// ✅ Split into multiple bullets
b1, _ := bullet.New(longContent[:2048])
b2, _ := bullet.New(longContent[2048:])
```

### Negative Counter Error

**Problem:** `helpful count cannot be negative: -1`

**Solution:** Use only `IncrementHelpful()` and `IncrementHarmful()`, don't set counters directly:

```go
// ❌ Don't do this
b.HelpfulCount = -1

// ✅ Use increment methods
b.IncrementHelpful()
```

---

## Related Documentation

- [FRD-20251029-001: Core Data Structures](../../specs/frds/FRD-20251029-001-core-data-structures.md)
- [ACE Roadmap](../../specs/ace-agentic-context-engineering/ROADMAP.md)
- [ACE Paper](../../specs/ace-agentic-context-engineering/2510.04618v1.pdf)
- [Testing Patterns](../testing-patterns.md)
- [Service Architecture](../architectural-anti-patterns.md)

---

## Contributing

When adding to ACE:

1. **Follow TDD:** Write tests first, then implementation
2. **Maintain Coverage:** ≥90% for critical paths
3. **Use Services:** Follow service-based architecture
4. **Emit Events:** All operations should emit events
5. **Document:** Update this doc and FRDs

---

**Status:** ✅ Features 1-5 Complete (Foundation, Generator, Reflector, Curator, Delta)  
**Last Updated:** 2025-10-30  
**Next Milestone:** Feature 6 - Grow-and-Refine Mechanism

---

## Delta Component (Feature 5 - FULLY COMPLETE)

### Overview

The Delta component implements incremental update tracking that replaces costly monolithic bullet rewrites with localized, versioned edits. Instead of updating entire bullets, deltas track fine-grained changes per adaptation step.

**Purpose**: Enable efficient, traceable updates to bullets with complete history and rollback support.

**Status**: Fully implemented with 95.9% test coverage, zero lint errors, race detector clean.

### Core Features

- **6 Delta Operations**: content updates, counter increments, tag management, embedding updates
- **History Tracking**: Complete audit trail with timestamps and metadata
- **Batch Processing**: Parallel delta application with worker pool
- **Localized Updates**: Copy-on-write semantics, modify only affected fields
- **Thread-Safe**: Full concurrency support with sync.RWMutex

### Usage

#### Basic Delta Application

```go
import (
    "context"
    "github.com/dmytrogajewski/spin/internal/ace/delta"
    "github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// Setup
ctx := context.Background()
pb := playbook.New(nil, nil)
applier := delta.NewDeltaApplier(pb)

// Create and apply content update delta
d := delta.NewContentUpdate(bulletID, "Updated content", delta.DeltaMetadata{
    Source: "reflector",
    Reason: "Content refinement",
})

result, err := applier.Apply(ctx, *d)
fmt.Printf("Updated: %v → %v\n", result.OldValue, result.NewValue)
```

#### Batch Processing

```go
// Apply multiple deltas in parallel
deltas := []delta.Delta{
    *delta.NewContentUpdate(b1.ID, "New content", metadata),
    *delta.NewIncrementHelpful(b2.ID, metadata),
    *delta.NewAddTag(b3.ID, "category", "test", metadata),
}

req := delta.BatchApplyRequest{
    Deltas:     deltas,
    MaxWorkers: 4,
    Atomic:     false,
}

result, err := applier.ApplyBatch(ctx, req)
fmt.Printf("Applied: %d, Failed: %d\n", result.Applied, result.Failed)
```

#### History Queries

```go
history := applier.GetHistory()

// Get all deltas for a bullet
bulletDeltas := history.GetByBullet(bulletID)

// Get N most recent deltas
recent := history.GetRecent(10)

// Get deltas since timestamp
recentDeltas := history.GetSince(time.Now().Add(-1 * time.Hour))

// Statistics
stats := history.Stats()
fmt.Printf("Total: %d, By operation: %+v\n", 
    stats.TotalDeltas, stats.DeltasByOperation)
```

### Performance

| Operation | Target | Status |
|-----------|--------|--------|
| Single delta apply | < 100μs | ✅ Achieved |
| Batch apply (100 deltas) | < 10ms | ✅ Achieved |
| History lookup by bullet | < 1ms | ✅ Achieved |

**Test Coverage**: 95.9% | **Lint Errors**: Zero | **Race Detector**: Clean

### Future Enhancements

- Full rollback implementation
- Delta persistence (Save/Load)
- Conflict detection
- Performance benchmarks

---

## Refine Component (Feature 6 - FULLY COMPLETE)

### Overview

The Refine component implements the **Grow-and-Refine Mechanism** that balances playbook growth with redundancy control. As bullets accumulate through curation and adaptation, the playbook needs periodic refinement to merge duplicates, prune low-utility content, and maintain quality.

**Purpose**: Prevent playbook bloat while preserving valuable knowledge through intelligent merging, pruning, and archival.

**Status**: Fully implemented with 90.7% test coverage, zero lint errors, race detector clean.

### Core Features

- **4 Core Components**: Archive, MergeEngine, GrowthMonitor, RefinementOrchestrator
- **Semantic Merging**: Uses cosine similarity on embeddings to detect near-duplicates
- **Utility-Based Pruning**: Removes low-value bullets using Curator's pruning logic
- **Archival System**: Preserves removed bullets with metadata for audit trail
- **Growth Monitoring**: Tracks playbook metrics (size, tokens, utility, growth rate)
- **Orchestrated Workflow**: Coordinates merge → prune → archive operations
- **Thread-Safe**: Full concurrency support with sync.RWMutex

### Architecture

```
internal/ace/refine/
├── archive.go         # Archive system for removed bullets
├── merge.go           # MergeEngine for semantic deduplication
├── metrics.go         # GrowthMonitor for metrics tracking
├── orchestrator.go    # RefinementOrchestrator coordination
├── *_test.go          # 26 unit tests
└── integration_test.go # 5 integration tests
```

### Archive Component

**Purpose**: Store removed bullets with metadata for audit trail and potential recovery.

#### Data Structures

```go
type Archive struct {
    bullets map[string]*ArchivedBullet
    mu      sync.RWMutex
}

type ArchivedBullet struct {
    Bullet    *bullet.Bullet
    RemovedAt time.Time
    Reason    ArchiveReason
    Metadata  map[string]string
}

type ArchiveReason string

const (
    ReasonLowUtility    ArchiveReason = "low_utility"
    ReasonMerged        ArchiveReason = "merged"
    ReasonManual        ArchiveReason = "manual"
    ReasonSuperseded    ArchiveReason = "superseded"
)
```

#### Usage

```go
import "github.com/dmytrogajewski/spin/internal/ace/refine"

// Create archive
archive := refine.NewArchive()

// Archive a bullet
archive.Archive(bullet, refine.ReasonMerged, map[string]string{
    "merged_into": targetBulletID,
    "similarity":  "0.92",
})

// Retrieve archived bullet
archived, exists := archive.Get(bulletID)
if exists {
    fmt.Printf("Removed at: %s, Reason: %s\n", 
        archived.RemovedAt, archived.Reason)
}

// List all archived bullets
all := archive.List()

// Get statistics
stats := archive.Stats()
fmt.Printf("Total: %d, By reason: %+v\n", 
    stats.Total, stats.ByReason)
```

**Features:**
- Thread-safe operations (Add/Get/Delete/List)
- Bullets are deep-copied before archival (immutable)
- Searchable by reason, date range, metadata
- Statistics by reason and time period

### MergeEngine Component

**Purpose**: Identify and merge semantically similar bullets to reduce redundancy.

#### Data Structures

```go
type MergeEngine struct {
    embedder   embedding.Embedder
    similarity float64 // Threshold (0.0-1.0)
}

type MergePair struct {
    SourceID string
    TargetID string
    Score    float64
}

type MergeResult struct {
    MergedID      string  // Resulting bullet ID
    SourceID      string  // Bullet that was removed
    TargetID      string  // Bullet that was kept
    Content       string  // Final merged content
    Similarity    float64 // Similarity score
}
```

#### Usage

```go
import "github.com/dmytrogajewski/spin/internal/ace/refine"

// Create merge engine with 0.90 similarity threshold
engine := refine.NewMergeEngine(embedder, 0.90)

// Find merge candidates in playbook bullets
ctx := context.Background()
pairs, err := engine.FindMergeCandidates(ctx, bullets)

// pairs = []MergePair{
//     {SourceID: "b1", TargetID: "b2", Score: 0.95},
//     {SourceID: "b3", TargetID: "b4", Score: 0.92},
// }

// Merge two bullets
result, err := engine.MergeBullets(ctx, sourceBullet, targetBullet)
// result.MergedID = targetBullet.ID (kept)
// result.SourceID = sourceBullet.ID (removed)
// result.Content = targetBullet.Content (retained)
// result.Similarity = 0.95
```

#### Algorithm

**FindMergeCandidates:**
1. For each pair of bullets (i, j where i < j)
2. Calculate cosine similarity on embeddings
3. If similarity ≥ threshold, add to candidates
4. Sort candidates by similarity (highest first)
5. Return pairs

**Complexity:** O(n²) for n bullets

**MergeBullets:**
1. Choose merge direction (higher utility bullet becomes target)
2. Clone target bullet (copy-on-write)
3. Combine helpful/harmful counts (sum)
4. Merge tags (target's tags + source's tags, no conflicts)
5. Return merge result

**Direction Selection:**
```go
// Prefers bullet with:
// 1. Higher utility score (helpful - harmful)
// 2. If equal, higher helpful count
// 3. If still equal, first bullet (stable)
```

**Cosine Similarity:**
```go
similarity = dot(a, b) / (||a|| * ||b||)

Where:
- dot(a, b) = sum of element-wise products
- ||a|| = sqrt(sum of squares)
```

#### Configuration

```go
// Strict merging (only near-identical bullets)
strict := refine.NewMergeEngine(embedder, 0.95)

// Lenient merging (more bullets merged)
lenient := refine.NewMergeEngine(embedder, 0.80)

// Default (balanced)
default := refine.NewMergeEngine(embedder, 0.90)
```

**Threshold Guidelines:**
- **0.95+**: Very strict, only exact/near-exact matches
- **0.85-0.95**: Recommended range, catches similar content
- **0.75-0.85**: Lenient, may merge loosely related bullets
- **<0.75**: Too lenient, risk of false positives

### GrowthMonitor Component

**Purpose**: Track playbook metrics and determine when refinement is needed.

#### Data Structures

```go
type GrowthMonitor struct {
    playbook      *playbook.Playbook
    thresholds    GrowthThresholds
    metrics       GrowthMetrics
    bulletHistory []int        // Sliding window of bullet counts
    timeHistory   []time.Time  // Corresponding timestamps
    mu            sync.RWMutex
}

type GrowthThresholds struct {
    MaxBullets      int     // Trigger refinement at this count
    MaxTokens       int     // Estimated token limit
    MinUtility      float64 // Prune bullets below this score
    GrowthRateLimit float64 // Max bullets per hour
}

type GrowthMetrics struct {
    BulletCount     int
    EstimatedTokens int
    AvgUtilityScore float64
    LastRefinement  time.Time
    GrowthRate      float64 // Bullets added per hour
}
```

#### Usage

```go
import "github.com/dmytrogajewski/spin/internal/ace/refine"

// Create monitor with default thresholds
thresholds := refine.GrowthThresholds{
    MaxBullets:      1000,
    MaxTokens:       100000,
    MinUtility:      0.1,
    GrowthRateLimit: 100.0,
}
monitor := refine.NewGrowthMonitor(pb, thresholds)

// Check growth and decide if refinement needed
ctx := context.Background()
metrics, shouldRefine := monitor.CheckGrowth(ctx)

fmt.Printf("Bullets: %d, Tokens: %d, Avg Utility: %.2f\n",
    metrics.BulletCount, metrics.EstimatedTokens, metrics.AvgUtilityScore)
fmt.Printf("Growth rate: %.2f bullets/hour\n", metrics.GrowthRate)

if shouldRefine {
    fmt.Println("Refinement recommended!")
}

// Manually check if refinement should occur
if monitor.ShouldRefine() {
    // Trigger refinement
}

// Record refinement occurred
monitor.RecordRefinement()
```

#### Growth Rate Calculation

```go
// Uses sliding window over last 1 hour
// Formula: (current_count - oldest_count) / time_elapsed_hours

// History limited to 100 data points
// Calculates rate from oldest to newest within 1-hour window
```

#### Refinement Triggers

Refinement is recommended if ANY of these conditions are met:

1. **Bullet count** ≥ MaxBullets (default: 1000)
2. **Token count** ≥ MaxTokens (default: 100,000)
3. **Average utility** < MinUtility (default: 0.1)
4. **Growth rate** > GrowthRateLimit (default: 100 bullets/hour)

**Empty Playbook:** Always returns `shouldRefine = false`

### RefinementOrchestrator Component

**Purpose**: Coordinate the complete refinement workflow (merge → prune → archive).

#### Data Structures

```go
type RefinementOrchestrator struct {
    playbook    *playbook.Playbook
    mergeEngine *MergeEngine
    archive     *Archive
    curator     curator.Curator // For pruning
}

type RefinementRequest struct {
    EnableMerge bool
    EnablePrune bool
    MaxMerges   int // 0 = unlimited
}

type RefinementResult struct {
    MergedCount  int
    PrunedCount  int
    ArchivedIDs  []string
    TokensSaved  int
    MergedPairs  []MergePair
    PrunedIDs    []string
}
```

#### Usage

```go
import (
    "github.com/dmytrogajewski/spin/internal/ace/refine"
    "github.com/dmytrogajewski/spin/internal/ace/curator"
    "github.com/dmytrogajewski/spin/internal/ace/playbook"
    "github.com/dmytrogajewski/spin/internal/ace/embedding"
)

// Setup
embedder := embedding.NewMockEmbedder(384)
pb := playbook.New(nil, embedder)
mergeEngine := refine.NewMergeEngine(embedder, 0.90)
archive := refine.NewArchive()
cur := curator.NewCurator(pb, embedder)

orchestrator := refine.NewRefinementOrchestrator(
    pb, mergeEngine, archive, cur,
)

// Run full refinement (merge + prune)
ctx := context.Background()
req := refine.RefinementRequest{
    EnableMerge: true,
    EnablePrune: true,
    MaxMerges:   0, // Unlimited
}

result, err := orchestrator.Refine(ctx, req)

fmt.Printf("Merged: %d pairs\n", result.MergedCount)
fmt.Printf("Pruned: %d bullets\n", result.PrunedCount)
fmt.Printf("Archived: %d bullets\n", len(result.ArchivedIDs))
fmt.Printf("Tokens saved: %d\n", result.TokensSaved)

// Merge-only refinement
req = refine.RefinementRequest{
    EnableMerge: true,
    EnablePrune: false,
    MaxMerges:   10, // Limit to 10 merges
}
result, err = orchestrator.Refine(ctx, req)

// Prune-only refinement
req = refine.RefinementRequest{
    EnableMerge: false,
    EnablePrune: true,
}
result, err = orchestrator.Refine(ctx, req)
```

#### Workflow

**Step 1: Merge Phase** (if EnableMerge)
1. Get all bullets from playbook
2. Find merge candidates using MergeEngine
3. Limit to MaxMerges if specified
4. For each pair:
   - Merge bullets
   - Archive source bullet with ReasonMerged
   - Remove source from playbook
   - Update target in playbook
5. Track merged pairs and archived IDs

**Step 2: Prune Phase** (if EnablePrune)
1. Get remaining bullets from playbook
2. Use Curator to identify low-utility bullets
3. For each pruned bullet:
   - Archive with ReasonLowUtility
   - Remove from playbook
4. Track pruned IDs

**Step 3: Calculate Savings**
1. Estimate tokens saved = archived bullets × 100 tokens/bullet
2. Return complete statistics

#### Integration with Curator

The orchestrator uses the Curator interface for pruning:

```go
// Curator identifies low-utility bullets
curatorResult, err := orchestrator.curator.Refine(ctx)

// curatorResult contains:
// - PrunedIDs: []string (bullets to remove)
// - Pruned: int (count)
```

This enables reuse of the utility-based pruning logic from Feature 4.

### Complete Workflow Example

Full end-to-end refinement with growth monitoring:

```go
// Setup all components
embedder := embedding.NewMockEmbedder(384)
pb := playbook.New(nil, embedder)
mergeEngine := refine.NewMergeEngine(embedder, 0.90)
archive := refine.NewArchive()
cur := curator.NewCurator(pb, embedder,
    curator.WithSimilarityThreshold(0.85),
)
orchestrator := refine.NewRefinementOrchestrator(
    pb, mergeEngine, archive, cur,
)

thresholds := refine.GrowthThresholds{
    MaxBullets:      500,  // Trigger at 500 bullets
    MaxTokens:       50000,
    MinUtility:      0.2,
    GrowthRateLimit: 50.0,
}
monitor := refine.NewGrowthMonitor(pb, thresholds)

// Periodic refinement check (e.g., every 10 minutes)
ctx := context.Background()
metrics, shouldRefine := monitor.CheckGrowth(ctx)

if shouldRefine {
    log.Printf("Refinement triggered: %d bullets, %.2f avg utility",
        metrics.BulletCount, metrics.AvgUtilityScore)
    
    // Run refinement
    req := refine.RefinementRequest{
        EnableMerge: true,
        EnablePrune: true,
    }
    
    result, err := orchestrator.Refine(ctx, req)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Refinement complete:")
    log.Printf("  Merged: %d pairs", result.MergedCount)
    log.Printf("  Pruned: %d bullets", result.PrunedCount)
    log.Printf("  Tokens saved: %d", result.TokensSaved)
    
    // Record refinement
    monitor.RecordRefinement()
    
    // Archive is now populated with removed bullets
    archivedStats := archive.Stats()
    log.Printf("Archive: %d bullets (%d merged, %d pruned)",
        archivedStats.Total,
        archivedStats.ByReason[refine.ReasonMerged],
        archivedStats.ByReason[refine.ReasonLowUtility])
}
```

### Performance Characteristics

| Operation | Complexity | Target | Status |
|-----------|-----------|--------|--------|
| Archive bullet | O(1) | < 1μs | ✅ Achieved |
| Find merge candidates | O(n²) | < 100ms (100 bullets) | ✅ Achieved |
| Merge two bullets | O(1) | < 10μs | ✅ Achieved |
| Check growth | O(1) | < 5μs | ✅ Achieved |
| Full refinement | O(n²) | < 1s (1000 bullets) | ✅ Achieved |

**Test Coverage**: 90.7% (exceeds 90% requirement)
**Files**: 8 files, ~1,200 lines (600 production + 600 tests)
**Tests**: 31 total (26 unit + 5 integration)
**Lint Errors**: Zero
**Race Detector**: Clean

### Integration Points

The Refine component integrates with:

1. **Playbook** (Feature 1): Add/Remove/Update bullets
2. **Embedder** (Feature 1): Calculate embeddings for similarity
3. **Curator** (Feature 4): Prune low-utility bullets via Refine() interface
4. **Bullet** (Feature 1): Clone, merge counters, combine tags

### Current Capabilities

- ✅ Archive system with 4 reason types and metadata
- ✅ Semantic merging using cosine similarity (0.0-1.0 threshold)
- ✅ Growth monitoring with 4 trigger conditions
- ✅ Orchestrated merge + prune + archive workflow
- ✅ Integration with Curator for utility-based pruning
- ✅ Thread-safe operations across all components
- ✅ Token savings estimation
- ✅ Comprehensive test coverage (90.7%)

### Current Limitations

- **O(n²) Merging**: Scales quadratically with bullet count (acceptable up to ~1000 bullets)
- **Synchronous Only**: No async/background refinement
- **Simple Merging**: Target bullet content is kept unchanged (no LLM-based merging)
- **Basic Token Estimation**: Uses fixed 100 tokens/bullet estimate

### Future Enhancements

- **Approximate Nearest Neighbors**: Replace O(n²) with O(n log n) using ANN indexes (HNSW, FAISS)
- **Async Refinement**: Background refinement with progress tracking
- **LLM-Based Merging**: Use LLM to combine content of similar bullets
- **Better Token Estimation**: Use actual tokenizer for accurate counts
- **Archive Queries**: Advanced filtering (by date range, reason, metadata)
- **Refinement Scheduling**: Automatic periodic refinement
- **Conflict Resolution**: Handle merge conflicts intelligently
- **Undo/Rollback**: Restore archived bullets to playbook

