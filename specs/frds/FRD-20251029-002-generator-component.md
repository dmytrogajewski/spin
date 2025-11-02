# FRD-20251029-002: Generator Component

**Feature:** Generator Component  
**Roadmap:** ACE (Agentic Context Engineering) - Feature 2  
**Status:** Draft  
**Created:** 2025-10-29  
**Author:** Spin Agent  
**Dependencies:** Feature 1 (Core Data Structures) - COMPLETED

---

## 1. Executive Summary

This FRD defines the Generator component for ACE (Agentic Context Engineering). The Generator produces reasoning trajectories that highlight which context bullets were useful or misleading during problem-solving. This enables the ACE system to learn from execution and refine contexts based on actual performance.

**Key Objectives:**
- Integrate context bullets into LLM prompts
- Generate reasoning trajectories with bullet feedback
- Track which bullets help or harm performance
- Support both labeled and unlabeled learning modes
- Enable itemized learning (IL) for detailed feedback extraction

---

## 2. Background

### 2.1 Problem Statement

From the ACE paper:

1. **Context Utility Tracking**: Need to know which context bullets actually help vs. harm performance
2. **Feedback Collection**: Must extract feedback from both labeled data (with ground truth) and unlabeled execution (without ground truth)
3. **Itemized Learning**: Need to mark specific bullets as helpful/harmful during trajectory generation
4. **Fine-Grained Retrieval**: Must select relevant bullets for each query without overwhelming context window

### 2.2 Solution Approach

The Generator implements three key workflows:

1. **ItemizedLearning (IL)**: Retrieve relevant bullets, inject into prompt, ask LLM to mark each bullet as helpful/harmful
2. **ContextBulletGenerator**: Generate new bullet candidates from task descriptions or execution traces
3. **TrajectoryGeneration**: Execute tasks with context bullets and collect execution feedback

### 2.3 Integration with Spin

Spin already has:
- **LLM Provider Interface** (`internal/llm/provider.go`) - unified interface for OpenAI, Ollama, LMStudio
- **Agent Loop** (`internal/agent/loop.go`) - multi-turn execution with tool calls
- **Message Handling** (`internal/agent/agent.go`) - conversation management
- **Tool Orchestration** (`internal/tools/`) - tool execution and result collection

ACE Generator will add:
- **Context Injection**: Insert relevant bullets into prompts
- **Feedback Extraction**: Parse bullet utility markers from responses
- **Trajectory Capture**: Record execution traces with bullet annotations
- **Bullet Generation**: Create new bullets from insights

---

## 3. Requirements

### 3.1 Functional Requirements

#### FR-1: Context Retrieval and Injection
**Priority:** P0 (Critical)

The Generator must:
- Retrieve relevant bullets from playbook based on query
- Construct system prompt with bullet list
- Format bullets with unique markers (e.g., `[B1]`, `[B2]`)
- Include instructions for feedback marking
- Inject into LLM request as system message

**Acceptance Criteria:**
- [ ] Semantic search retrieves top-K relevant bullets
- [ ] Bullets formatted with ID markers in prompt
- [ ] System prompt includes feedback instructions
- [ ] Context injection preserves message order
- [ ] Supports both zero-shot and few-shot examples

#### FR-2: Itemized Learning (IL)
**Priority:** P0 (Critical)

ItemizedLearning workflow:
1. Retrieve relevant bullets for task
2. Construct prompt with bullets and feedback instructions
3. Execute task with LLM
4. Parse response for bullet markers
5. Update bullet counters (helpful/harmful)
6. Return execution result

**Acceptance Criteria:**
- [ ] IL prompt constructed with bullets and instructions
- [ ] LLM response parsed for `HELPFUL: [B1, B3]` markers
- [ ] LLM response parsed for `HARMFUL: [B2]` markers
- [ ] Playbook updated with incremented counters
- [ ] Supports both labeled and unlabeled modes
- [ ] Returns trajectory with annotations

#### FR-3: Context Bullet Generation
**Priority:** P0 (Critical)

The Generator must create new bullets from:
- Task descriptions (pre-execution)
- Execution traces (post-execution)
- User feedback (explicit)
- Error patterns (from failures)

**Acceptance Criteria:**
- [ ] Generate bullets from natural language input
- [ ] Extract bullet candidates from trajectories
- [ ] Validate generated bullets (length, format)
- [ ] Tag generated bullets with metadata (source, timestamp)
- [ ] Support batch generation (multiple bullets at once)

#### FR-4: Trajectory Capture
**Priority:** P1 (High)

Capture execution traces with:
- Input query/task
- Retrieved context bullets
- LLM reasoning steps
- Tool calls and results
- Final output
- Success/failure indicators
- Bullet utility annotations

**Acceptance Criteria:**
- [ ] Trajectory includes all execution steps
- [ ] Tool calls recorded with arguments and results
- [ ] Bullet usage tracked per step
- [ ] Success/failure determined from execution
- [ ] Trajectory serializable to JSON
- [ ] Timestamped for temporal analysis

### 3.2 Non-Functional Requirements

#### NFR-1: Performance
- Context retrieval: < 10ms for 1000 bullets
- Bullet injection: < 5ms overhead per request
- Feedback parsing: < 1ms per response
- Trajectory serialization: < 10ms per trajectory

#### NFR-2: Scalability
- Support 100+ bullets in single prompt (within token limits)
- Handle 1000+ trajectories in memory
- Batch processing: 10+ tasks in parallel

#### NFR-3: Reliability
- Graceful degradation if bullet retrieval fails (continue without context)
- Robust parsing (handle malformed feedback markers)
- Validation of generated bullets
- Error recovery in multi-turn loops

#### NFR-4: Observability
- Emit events for all operations (retrieval, injection, parsing)
- Log bullet usage statistics
- Track retrieval quality metrics
- Monitor generation success rates

---

## 4. Architecture

### 4.1 Package Structure

```
internal/ace/
├── generator/
│   ├── generator.go           # Main Generator interface
│   ├── itemized_learning.go   # ItemizedLearning implementation
│   ├── bullet_gen.go          # Bullet generation
│   ├── trajectory.go          # Trajectory data structures
│   ├── prompt.go              # Prompt construction
│   ├── parser.go              # Feedback parsing
│   ├── generator_test.go      # Unit tests
│   └── integration_test.go    # Integration tests
│
├── retrieval/
│   ├── retriever.go           # Retrieval interface
│   ├── semantic.go            # Semantic search retriever
│   ├── hybrid.go              # Hybrid retrieval (semantic + keyword)
│   └── retriever_test.go
│
└── feedback/
    ├── feedback.go            # Feedback data structures
    ├── parser.go              # Feedback parser
    ├── extractor.go           # Execution feedback extraction
    └── feedback_test.go
```

### 4.2 Core Interfaces

#### Generator Interface

```go
package generator

import (
    "context"
    "github.com/dmytrogajewski/spin/internal/ace/bullet"
    "github.com/dmytrogajewski/spin/internal/ace/playbook"
    "github.com/dmytrogajewski/spin/internal/llm"
)

// Generator produces reasoning trajectories with context bullets.
type Generator interface {
    // ItemizedLearning retrieves bullets, injects into prompt, executes task,
    // and collects feedback on bullet utility.
    ItemizedLearning(ctx context.Context, req ItemizedLearningRequest) (*ItemizedLearningResponse, error)
    
    // GenerateBullets creates new bullet candidates from input.
    GenerateBullets(ctx context.Context, req BulletGenerationRequest) ([]*bullet.Bullet, error)
    
    // GenerateTrajectory executes a task with context bullets and captures full trace.
    GenerateTrajectory(ctx context.Context, req TrajectoryRequest) (*Trajectory, error)
}

// ItemizedLearningRequest is input for ItemizedLearning workflow.
type ItemizedLearningRequest struct {
    // Query is the task description or question
    Query string
    
    // GroundTruth is the expected answer (optional, for labeled learning)
    GroundTruth string
    
    // TopK is number of bullets to retrieve
    TopK int
    
    // Model is the LLM model to use
    Model string
    
    // Temperature controls randomness
    Temperature float64
    
    // MaxTokens limits response length
    MaxTokens int
}

// ItemizedLearningResponse is output from ItemizedLearning.
type ItemizedLearningResponse struct {
    // Trajectory is the full execution trace
    Trajectory *Trajectory
    
    // Feedback contains bullet utility annotations
    Feedback *BulletFeedback
    
    // Output is the final answer
    Output string
    
    // Success indicates if task succeeded
    Success bool
}

// BulletGenerationRequest is input for bullet generation.
type BulletGenerationRequest struct {
    // Input is the source text (task, trajectory, feedback)
    Input string
    
    // SourceType indicates input type ("task", "trajectory", "feedback")
    SourceType string
    
    // MaxBullets limits number of generated bullets
    MaxBullets int
    
    // Tags to apply to generated bullets
    Tags map[string]string
}

// TrajectoryRequest is input for trajectory generation.
type TrajectoryRequest struct {
    // Query is the task description
    Query string
    
    // Messages is the conversation history (optional)
    Messages []Message
    
    // RetrievedBullets are context bullets to inject
    RetrievedBullets []*bullet.Bullet
    
    // Tools available for execution (optional)
    Tools []Tool
    
    // MaxTurns limits agent loop iterations
    MaxTurns int
}

// Message represents a conversation message.
type Message struct {
    Role      string    // "user", "assistant", "system", "tool"
    Content   string
    Timestamp time.Time
}

// Tool represents an available tool.
type Tool struct {
    Name        string
    Description string
    Parameters  json.RawMessage
}
```

#### Trajectory Data Structures

```go
package generator

import (
    "time"
    "github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// Trajectory is a complete execution trace.
type Trajectory struct {
    // ID is unique identifier
    ID string
    
    // Query is the input task
    Query string
    
    // RetrievedBullets are context bullets used
    RetrievedBullets []*bullet.Bullet
    
    // Steps are execution steps in order
    Steps []TrajectoryStep
    
    // Output is the final result
    Output string
    
    // Success indicates if task succeeded
    Success bool
    
    // BulletFeedback contains utility annotations
    BulletFeedback *BulletFeedback
    
    // Metadata contains additional info
    Metadata TrajectoryMetadata
    
    // CreatedAt is when trajectory was generated
    CreatedAt time.Time
}

// TrajectoryStep is a single reasoning or execution step.
type TrajectoryStep struct {
    // StepNumber is the step index (0-based)
    StepNumber int
    
    // Type is step type ("reasoning", "tool_call", "tool_result")
    Type string
    
    // Content is the step content
    Content string
    
    // ToolCall is tool execution details (if Type == "tool_call")
    ToolCall *ToolCall
    
    // ToolResult is tool output (if Type == "tool_result")
    ToolResult *ToolResult
    
    // Timestamp is when step occurred
    Timestamp time.Time
}

// ToolCall represents a tool execution.
type ToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

// ToolResult represents tool output.
type ToolResult struct {
    ToolCallID string
    Output     string
    Error      string
    ExitCode   int
}

// TrajectoryMetadata contains additional trajectory info.
type TrajectoryMetadata struct {
    Model       string
    Temperature float64
    MaxTokens   int
    TotalTokens int
    Duration    time.Duration
    Turns       int
}

// BulletFeedback contains utility annotations for bullets.
type BulletFeedback struct {
    // HelpfulBullets are bullets marked as helpful
    HelpfulBullets []string // Bullet IDs
    
    // HarmfulBullets are bullets marked as harmful
    HarmfulBullets []string // Bullet IDs
    
    // Explanation is optional reasoning for feedback
    Explanation string
}
```

#### Retriever Interface

```go
package retrieval

import (
    "context"
    "github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// Retriever retrieves relevant bullets for a query.
type Retriever interface {
    // Retrieve finds top-K relevant bullets for query.
    Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error)
    
    // RetrieveWithScores returns bullets with relevance scores.
    RetrieveWithScores(ctx context.Context, query string, topK int) ([]ScoredBullet, error)
}

// ScoredBullet is a bullet with relevance score.
type ScoredBullet struct {
    Bullet *bullet.Bullet
    Score  float64  // Relevance score (0.0 to 1.0)
}

// SemanticRetriever uses embeddings for retrieval.
type SemanticRetriever struct {
    playbook *playbook.Playbook
    embedder embedding.Embedder
}

// NewSemanticRetriever creates a semantic retriever.
func NewSemanticRetriever(pb *playbook.Playbook, emb embedding.Embedder) *SemanticRetriever

// Retrieve implements Retriever interface.
func (r *SemanticRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error)
```

#### Prompt Construction

```go
package generator

// PromptBuilder constructs prompts with context bullets.
type PromptBuilder struct {
    systemPrompt string
    includeIL    bool  // Include ItemizedLearning instructions
}

// NewPromptBuilder creates a prompt builder.
func NewPromptBuilder(opts ...PromptOption) *PromptBuilder

// PromptOption configures PromptBuilder.
type PromptOption func(*PromptBuilder)

// WithSystemPrompt sets custom system prompt.
func WithSystemPrompt(prompt string) PromptOption

// WithItemizedLearning enables IL instructions.
func WithItemizedLearning() PromptOption

// BuildSystemPrompt constructs system prompt with bullets.
func (pb *PromptBuilder) BuildSystemPrompt(bullets []*bullet.Bullet) string

// FormatBullet formats a bullet with marker for IL.
func (pb *PromptBuilder) FormatBullet(b *bullet.Bullet, index int) string
```

#### Feedback Parser

```go
package feedback

import "github.com/dmytrogajewski/spin/internal/ace/generator"

// Parser extracts bullet feedback from LLM responses.
type Parser interface {
    // Parse extracts BulletFeedback from response text.
    Parse(response string) (*generator.BulletFeedback, error)
}

// RegexParser uses regex patterns to extract feedback.
type RegexParser struct {
    helpfulPattern *regexp.Regexp
    harmfulPattern *regexp.Regexp
}

// NewRegexParser creates a regex-based parser.
func NewRegexParser() *RegexParser

// Parse implements Parser interface.
func (p *RegexParser) Parse(response string) (*generator.BulletFeedback, error)
```

### 4.3 Implementation Details

#### ItemizedLearning Workflow

```go
func (g *GeneratorImpl) ItemizedLearning(ctx context.Context, req ItemizedLearningRequest) (*ItemizedLearningResponse, error) {
    // 1. Retrieve relevant bullets
    bullets, err := g.retriever.Retrieve(ctx, req.Query, req.TopK)
    if err != nil {
        return nil, fmt.Errorf("retrieve bullets: %w", err)
    }
    
    // 2. Build system prompt with bullets and IL instructions
    systemPrompt := g.promptBuilder.BuildSystemPrompt(bullets)
    
    // 3. Construct messages
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage(systemPrompt),
        openai.UserMessage(req.Query),
    }
    
    // 4. Call LLM
    params := openai.ChatCompletionNewParams{
        Messages:    openai.F(messages),
        Model:       openai.F(openai.ChatModel(req.Model)),
        Temperature: openai.Float(req.Temperature),
        MaxTokens:   openai.Int(int64(req.MaxTokens)),
    }
    
    resp, err := g.llm.Complete(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("llm complete: %w", err)
    }
    
    // 5. Parse response
    output := resp.Choices[0].Message.Content
    
    // 6. Extract feedback
    feedback, err := g.feedbackParser.Parse(output)
    if err != nil {
        // Log error but don't fail - feedback is optional
        g.logger.Warn("failed to parse feedback", "error", err)
        feedback = &generator.BulletFeedback{}
    }
    
    // 7. Update playbook
    if err := g.updatePlaybook(ctx, feedback); err != nil {
        return nil, fmt.Errorf("update playbook: %w", err)
    }
    
    // 8. Build trajectory
    trajectory := g.buildTrajectory(req, bullets, resp, feedback)
    
    // 9. Determine success (if ground truth provided)
    success := false
    if req.GroundTruth != "" {
        success = g.checkSuccess(output, req.GroundTruth)
    }
    
    return &ItemizedLearningResponse{
        Trajectory: trajectory,
        Feedback:   feedback,
        Output:     output,
        Success:    success,
    }, nil
}
```

#### Prompt Template for ItemizedLearning

```go
const itemizedLearningTemplate = `You are an expert assistant. Below is a context playbook with strategies and domain knowledge.

# Context Playbook

{{range $i, $bullet := .Bullets}}
[B{{$i}}] {{$bullet.Content}}
{{end}}

# Instructions

1. Use the context bullets above to solve the task
2. After solving, indicate which bullets were helpful or harmful:
   - HELPFUL: [B1, B3, ...] - bullets that helped solve the task
   - HARMFUL: [B2, ...] - bullets that misled or were incorrect
3. Provide your reasoning

# Task

{{.Query}}

# Response Format

<your solution>

HELPFUL: [list of helpful bullet markers]
HARMFUL: [list of harmful bullet markers]
EXPLANATION: <brief explanation of feedback>
`
```

#### Feedback Parsing

```go
func (p *RegexParser) Parse(response string) (*generator.BulletFeedback, error) {
    feedback := &generator.BulletFeedback{
        HelpfulBullets: []string{},
        HarmfulBullets: []string{},
    }
    
    // Extract HELPFUL markers
    helpfulMatches := p.helpfulPattern.FindStringSubmatch(response)
    if len(helpfulMatches) > 1 {
        // Parse "[B1, B3, B5]" -> ["B1", "B3", "B5"]
        markers := parseBulletMarkers(helpfulMatches[1])
        for _, marker := range markers {
            bulletID := g.markerToBulletID(marker)
            feedback.HelpfulBullets = append(feedback.HelpfulBullets, bulletID)
        }
    }
    
    // Extract HARMFUL markers
    harmfulMatches := p.harmfulPattern.FindStringSubmatch(response)
    if len(harmfulMatches) > 1 {
        markers := parseBulletMarkers(harmfulMatches[1])
        for _, marker := range markers {
            bulletID := g.markerToBulletID(marker)
            feedback.HarmfulBullets = append(feedback.HarmfulBullets, bulletID)
        }
    }
    
    // Extract EXPLANATION (optional)
    explMatches := p.explanationPattern.FindStringSubmatch(response)
    if len(explMatches) > 1 {
        feedback.Explanation = strings.TrimSpace(explMatches[1])
    }
    
    return feedback, nil
}

// Pattern examples:
// helpfulPattern = regexp.MustCompile(`HELPFUL:\s*\[(.*?)\]`)
// harmfulPattern = regexp.MustCompile(`HARMFUL:\s*\[(.*?)\]`)
// explanationPattern = regexp.MustCompile(`EXPLANATION:\s*(.+?)(?:\n|$)`)
```

#### Bullet Generation

```go
func (g *GeneratorImpl) GenerateBullets(ctx context.Context, req BulletGenerationRequest) ([]*bullet.Bullet, error) {
    // Construct prompt based on source type
    var prompt string
    switch req.SourceType {
    case "task":
        prompt = fmt.Sprintf(taskBulletPrompt, req.Input, req.MaxBullets)
    case "trajectory":
        prompt = fmt.Sprintf(trajectoryBulletPrompt, req.Input, req.MaxBullets)
    case "feedback":
        prompt = fmt.Sprintf(feedbackBulletPrompt, req.Input, req.MaxBullets)
    default:
        return nil, fmt.Errorf("unknown source type: %s", req.SourceType)
    }
    
    // Call LLM
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage(bulletGenerationSystemPrompt),
        openai.UserMessage(prompt),
    }
    
    params := openai.ChatCompletionNewParams{
        Messages:    openai.F(messages),
        Model:       openai.F(openai.ChatModel(g.config.Model)),
        Temperature: openai.Float(0.7),  // Some creativity
        MaxTokens:   openai.Int(2000),
    }
    
    resp, err := g.llm.Complete(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("llm complete: %w", err)
    }
    
    // Parse response to extract bullet candidates
    output := resp.Choices[0].Message.Content
    candidates := g.parseBulletCandidates(output)
    
    // Validate and create bullets
    bullets := make([]*bullet.Bullet, 0, len(candidates))
    for _, content := range candidates {
        b, err := bullet.New(content, bullet.WithTags(req.Tags))
        if err != nil {
            g.logger.Warn("invalid bullet candidate", "content", content, "error", err)
            continue
        }
        bullets = append(bullets, b)
    }
    
    return bullets, nil
}

const bulletGenerationSystemPrompt = `You are an expert at distilling insights into concise, actionable strategies.
Extract concrete, specific strategies or domain knowledge from the input.
Each strategy should be:
- Actionable (not vague advice)
- Self-contained (understandable alone)
- Concise (< 200 chars preferred)
- Specific to the domain/task

Format: Output one strategy per line, numbered.`

const taskBulletPrompt = `Extract up to %d key strategies for solving this type of task:

%s`
```

---

## 5. Implementation Plan

### Phase 1: Core Interfaces and Data Structures (Days 1-2)

**TDD Micro-cycles:**

1. Test Trajectory struct creation and serialization
2. Test BulletFeedback creation and validation
3. Test Message struct and conversions
4. Test ToolCall and ToolResult structures

**Deliverables:**
- [ ] `trajectory.go` - Trajectory data structures
- [ ] `feedback/feedback.go` - Feedback structures
- [ ] Unit tests with 90%+ coverage

### Phase 2: Retrieval Component (Days 3-4)

**TDD Micro-cycles:**

1. Test Retriever interface with mock
2. Test SemanticRetriever with mock embedder
3. Test top-K retrieval correctness
4. Test RetrieveWithScores ordering
5. Test empty playbook edge case
6. Test query with no matches

**Deliverables:**
- [ ] `retrieval/retriever.go` - Retriever interface
- [ ] `retrieval/semantic.go` - Semantic retriever
- [ ] Unit tests with 90%+ coverage

### Phase 3: Prompt Construction (Day 5)

**TDD Micro-cycles:**

1. Test PromptBuilder creation
2. Test BuildSystemPrompt with no bullets
3. Test BuildSystemPrompt with bullets
4. Test FormatBullet with IL markers
5. Test template rendering
6. Test custom system prompts

**Deliverables:**
- [ ] `prompt.go` - PromptBuilder
- [ ] Unit tests with 90%+ coverage

### Phase 4: Feedback Parsing (Days 6-7)

**TDD Micro-cycles:**

1. Test RegexParser creation
2. Test parsing HELPFUL markers
3. Test parsing HARMFUL markers
4. Test parsing EXPLANATION
5. Test malformed feedback handling
6. Test empty feedback
7. Test combined HELPFUL + HARMFUL

**Deliverables:**
- [ ] `feedback/parser.go` - RegexParser
- [ ] Unit tests with 90%+ coverage

### Phase 5: ItemizedLearning Implementation (Days 8-10)

**TDD Micro-cycles:**

1. Test Generator creation with dependencies
2. Test ItemizedLearning with mock LLM
3. Test bullet retrieval step
4. Test prompt construction step
5. Test LLM call step
6. Test feedback parsing step
7. Test playbook update step
8. Test trajectory building
9. Test success determination (with ground truth)
10. Test error handling (retrieval fails)
11. Test error handling (LLM fails)
12. Test error handling (parsing fails)

**Deliverables:**
- [ ] `generator.go` - Generator interface
- [ ] `itemized_learning.go` - ItemizedLearning impl
- [ ] Unit tests with 90%+ coverage
- [ ] Integration test with real playbook

### Phase 6: Bullet Generation (Days 11-12)

**TDD Micro-cycles:**

1. Test GenerateBullets with "task" source
2. Test GenerateBullets with "trajectory" source
3. Test GenerateBullets with "feedback" source
4. Test bullet candidate parsing
5. Test validation of generated bullets
6. Test tagging of generated bullets
7. Test max bullets limit

**Deliverables:**
- [ ] `bullet_gen.go` - Bullet generation
- [ ] Unit tests with 90%+ coverage

### Phase 7: Trajectory Generation (Days 13-14)

**TDD Micro-cycles:**

1. Test GenerateTrajectory with simple query
2. Test trajectory step recording
3. Test tool call capture
4. Test tool result capture
5. Test multi-turn execution
6. Test metadata collection
7. Test success/failure detection

**Deliverables:**
- [ ] `trajectory.go` - Trajectory generation
- [ ] Unit tests with 90%+ coverage
- [ ] Integration test with agent loop

### Phase 8: Integration & Polish (Day 15)

**Tasks:**
1. Run full integration tests
2. Run `uast parse | herr analyze`
3. Run `make lint` and fix errors
4. Verify 90%+ coverage
5. Run race detector
6. Update documentation
7. Update ROADMAP.md

**Deliverables:**
- [ ] Zero lint errors
- [ ] Zero race conditions
- [ ] 90%+ coverage
- [ ] Integration tests passing
- [ ] Documentation complete

---

## 6. Testing Strategy

### 6.1 Unit Tests

**Coverage Target:** ≥90%

**Key Test Cases:**

#### Trajectory Tests
- Creation with all fields
- JSON serialization round-trip
- Step ordering verification
- Metadata accuracy

#### Feedback Tests
- BulletFeedback creation
- Validation (valid/invalid IDs)
- Empty feedback handling

#### Retrieval Tests
- Top-K retrieval accuracy
- Score ordering (descending)
- Empty playbook
- No matches for query
- Mock embedder integration

#### Prompt Tests
- System prompt construction
- Bullet formatting with markers
- IL instructions inclusion
- Custom system prompts
- Template rendering

#### Parser Tests
- HELPFUL marker extraction
- HARMFUL marker extraction
- EXPLANATION extraction
- Malformed feedback handling
- Edge cases (empty, partial)

#### ItemizedLearning Tests
- End-to-end workflow with mocks
- Each step in isolation
- Error paths (retrieval, LLM, parsing)
- Playbook update verification
- Success determination

#### BulletGeneration Tests
- Task-based generation
- Trajectory-based generation
- Feedback-based generation
- Candidate parsing
- Validation of generated bullets

### 6.2 Integration Tests

**Scope:** End-to-end workflows

**Test Scenarios:**

1. **ItemizedLearning Full Flow**
   - Create playbook with test bullets
   - Run IL with real query
   - Verify feedback extraction
   - Verify playbook updates
   - Check trajectory completeness

2. **Bullet Generation Pipeline**
   - Generate bullets from task
   - Add to playbook
   - Use in next IL iteration
   - Verify cumulative learning

3. **Multi-Turn Trajectory**
   - Execute task with tool calls
   - Capture all steps
   - Verify metadata
   - Check success detection

### 6.3 Performance Tests

```go
func BenchmarkRetrieve(b *testing.B)
func BenchmarkPromptBuild(b *testing.B)
func BenchmarkFeedbackParse(b *testing.B)
func BenchmarkItemizedLearning(b *testing.B)
```

**Target Performance:**
- Retrieval (1000 bullets): < 10ms
- Prompt build: < 5ms
- Feedback parse: < 1ms
- Full IL cycle: < 500ms (dominated by LLM call)

---

## 7. Success Criteria

### 7.1 Functional Success

- [ ] All functional requirements (FR-1 to FR-4) implemented
- [ ] All acceptance criteria met
- [ ] All unit tests passing
- [ ] All integration tests passing

### 7.2 Quality Success

- [ ] Test coverage ≥ 90%
- [ ] Zero lint errors (`make lint`)
- [ ] Zero race conditions (`go test -race`)
- [ ] GoDoc on all exports
- [ ] `uast/herr` analysis clean (at least YELLOW)

### 7.3 Performance Success

- [ ] Retrieval < 10ms for 1000 bullets
- [ ] Prompt build < 5ms
- [ ] Feedback parse < 1ms
- [ ] IL cycle < 500ms (excluding LLM)

### 7.4 Documentation Success

- [ ] FRD complete and reviewed
- [ ] GoDoc with examples
- [ ] Integration guide in `docs/packages/ace.md`
- [ ] ROADMAP.md updated

---

## 8. Risks and Mitigations

### Risk 1: LLM Feedback Unreliable
**Impact:** High  
**Probability:** Medium  

**Risk:** LLM may not consistently provide feedback markers in expected format

**Mitigation:**
- Use clear, explicit instructions in prompt
- Provide few-shot examples
- Implement robust regex parsing with fallbacks
- Make feedback optional (don't fail if parsing fails)
- Add validation and retry logic

### Risk 2: Context Window Overflow
**Impact:** Medium  
**Probability:** High  

**Risk:** Too many bullets exceed LLM context window

**Mitigation:**
- Implement dynamic top-K selection
- Monitor total token count
- Truncate bullets if needed
- Add warnings when approaching limits
- Support bullet prioritization

### Risk 3: Retrieval Quality
**Impact:** High  
**Probability:** Medium  

**Risk:** Semantic search may retrieve irrelevant bullets

**Mitigation:**
- Add relevance threshold filtering
- Support multiple retrieval strategies (semantic, keyword, hybrid)
- Log retrieval scores for analysis
- Allow manual bullet selection in tests
- Implement retrieval evaluation metrics

### Risk 4: Prompt Injection
**Impact:** Low  
**Probability:** Low  

**Risk:** Malicious bullet content could manipulate prompt

**Mitigation:**
- Validate bullet content on creation
- Sanitize special characters in prompts
- Use clear delimiters ([B1], [B2], etc.)
- Document security considerations

---

## 9. Future Enhancements

**Out of scope for this FRD:**

1. **Hybrid Retrieval** (Feature 3)
   - Combine semantic + keyword search
   - BM25 + embedding fusion

2. **Adaptive Top-K** (Feature 4)
   - Dynamically adjust bullet count based on task complexity
   - Token budget optimization

3. **Bullet Reranking** (Feature 5)
   - Rerank retrieved bullets by utility score
   - Use historical feedback for ranking

4. **Multi-Model Generation** (Feature 6)
   - Generate with multiple LLMs
   - Ensemble bullet generation

5. **Streaming Support** (Feature 7)
   - Stream trajectory steps in real-time
   - Progressive feedback collection

---

## 10. References

- **ACE Paper:** ACE (Agentic Context Engineering) specification
- **Feature 1:** FRD-20251029-001 (Core Data Structures) - COMPLETED
- **Roadmap:** `specs/ace-agentic-context-engineering/ROADMAP.md`
- **LLM Integration:** `internal/llm/provider.go`
- **Agent Loop:** `internal/agent/loop.go`
- **Testing Patterns:** `docs/testing-patterns.md`

---

## 11. Approval

**Author:** Spin Agent  
**Created:** 2025-10-29  
**Status:** Draft - Ready for Implementation  

**Checklist:**
- [x] Requirements complete and testable
- [x] Architecture designed
- [x] Integration with Spin analyzed
- [x] Test strategy defined
- [x] Success criteria clear
- [x] Risks identified and mitigated
- [x] Performance targets specified

---

## Appendix A: Example Prompts

### ItemizedLearning Prompt Example

```
You are an expert assistant. Below is a context playbook with strategies and domain knowledge.

# Context Playbook

[B1] Always validate user input before processing
[B2] Use parameterized queries to prevent SQL injection
[B3] Prefer composition over inheritance in Go
[B4] Use table-driven tests for repetitive test cases

# Instructions

1. Use the context bullets above to solve the task
2. After solving, indicate which bullets were helpful or harmful:
   - HELPFUL: [B1, B3, ...] - bullets that helped solve the task
   - HARMFUL: [B2, ...] - bullets that misled or were incorrect
3. Provide your reasoning

# Task

Write a Go function that validates and stores user comments in a database.

# Response Format

<your solution>

HELPFUL: [list of helpful bullet markers]
HARMFUL: [list of harmful bullet markers]
EXPLANATION: <brief explanation of feedback>
```

### Bullet Generation Prompt Example

```
Extract up to 5 key strategies for solving this type of task:

Write a concurrent web scraper in Go that respects rate limits and handles errors gracefully.

Output one strategy per line, numbered.
```

Expected response:
```
1. Use worker pool pattern with buffered channels for concurrency control
2. Implement exponential backoff for failed requests
3. Use context.Context for timeout and cancellation
4. Add rate limiter to respect robots.txt and avoid overwhelming servers
5. Separate concerns: fetcher, parser, and storage components
```

---

**End of FRD-20251029-002**
