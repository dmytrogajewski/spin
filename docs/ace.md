# ACE: Agentic Context Engineering

## NAME

ACE - autonomous learning system for AI agents through trajectory analysis and playbook maintenance

## SYNOPSIS

```yaml
# ~/.spin/config.yaml
ace:
  enabled: true
  playbook_path: ~/.spin/ace/playbook.json
  trajectory_path: ~/.spin/ace/trajectories/
  top_k: 5
  min_score: 0.7
```

## DESCRIPTION

ACE (Agentic Context Engineering) is a self-improving knowledge system that learns from agent execution trajectories. It maintains a playbook of learned strategies (bullets) and retrieves relevant knowledge during task execution.

ACE operates in two phases:
1. **Retrieval**: Progressive context-aware bullet retrieval during execution
2. **Learning**: Trajectory analysis and bullet generation after completion

Key characteristics:
- Semantic search using vector embeddings (Ollama nomic-embed-text)
- Progressive retrieval with adaptive triggers
- Dual-path learning: quick generation or reflection-based analysis
- Quality control through deduplication and refinement
- Delta-based change tracking for playbook evolution

## ARCHITECTURE

```
Agent Execution Loop
    │
    ├─ Progressive Retrieval (during execution)
    │   ├─ Trigger Detection → shouldRetrieveProgressive()
    │   ├─ Query Building → buildQueryFromContext()
    │   ├─ Semantic Search → Retriever.Retrieve()
    │   ├─ Cache Management → TrajectoryContext.BulletCache
    │   └─ Event Emission → EventACERetrieval
    │
    └─ Learning Pipeline (after completion)
        ├─ Trajectory Building → TrajectoryContext.ToTrajectory()
        │
        ├─ Path A: Quick Generation
        │   └─ Generator.GenerateBullets()
        │
        ├─ Path B: Reflection-based (AutoReflect=true)
        │   ├─ Reflector.Reflect() → extract insights
        │   └─ Curator.Curate() → deduplicate & merge
        │
        ├─ Playbook Update → delta operations
        └─ Event Emission → EventACELearned
```

## PROGRESSIVE RETRIEVAL

ACE retrieves relevant bullets during execution using four adaptive triggers:

### Trigger Types

```
TRIGGER          PRIORITY  CONDITION                      QUERY STRATEGY
TriggerInitial   1         turn == 0                      base query only
TriggerError     2         recent error detected          base + error patterns
TriggerToolChange 3        tool usage changed             base + tool names
TriggerInterval  4         cache TTL expired              base + concepts
```

Trigger evaluation (first match wins):
1. **Initial**: Always fires on turn 0
2. **Error**: Fires when errors detected in last N steps (configurable via `error_lookback`)
3. **ToolChange**: Fires when different tools used in last N steps (`tool_change_lookback`)
4. **Interval**: Fires when `current_turn - last_retrieval >= cache_ttl`

### Query Construction

Queries adapt based on trigger type:

```go
// TriggerInitial
query = "implement user authentication"

// TriggerError  
query = "implement user authentication" + 
        "JWT validation failed" + 
        "bcrypt hash error"

// TriggerToolChange
query = "implement user authentication" +
        "write_file" + "shell_command" + "read_file"

// TriggerInterval
query = "implement user authentication" +
        "password hashing" + "token generation"
```

### Caching Strategy

Bullets retrieved during execution are cached with TTL:

```
BulletCache: map[bulletID] → {bullet, retrievedAtTurn}

Active bullets: retrievedAtTurn + CacheTTL >= CurrentTurn
Expired bullets: dropped from context
```

Cache metrics tracked per trajectory:
- `CacheHits`: queries finding cached bullets
- `CacheMisses`: queries triggering new retrieval
- `CacheSize`: total unique bullets cached

### Retrieval Flow

```
Turn N starts
    ↓
shouldRetrieveProgressive() checks triggers
    ↓ (if triggered)
buildQueryFromContext() constructs query
    ↓
Retriever.Retrieve(query) performs semantic search
    ↓
TrajectoryContext.RecordRetrieval() caches bullets
    ↓
EventACERetrieval emitted (if EmitACEEvents=true)
    ↓
TUI displays: "⟐ Retrieved 3 new strategies:"
              "  • Use bcrypt for password hashing..."
              "  • Validate JWT signatures with..."
    ↓
GetActiveBullets() returns TTL-filtered bullets
    ↓
Bullets injected into LLM prompt context
```

### Configuration

```yaml
ace:
  retrieval:
    top_k: 5                    # Max bullets per retrieval
    min_score: 0.7              # Similarity threshold (0-1)
    progressive_context:
      enabled: true
      cache_ttl: 3              # Bullet lifetime in turns
      error_lookback: 3         # Steps to scan for errors
      tool_change_lookback: 2   # Steps to scan for tool changes
      emit_ace_events: true     # Show retrieval hints in TUI
```

## LEARNING PIPELINE

After execution completes (success or failure), ACE generates bullets from the trajectory.

### Trajectory Structure

```go
type Trajectory struct {
    ID      string          // Unique identifier
    Query   string          // Original user task
    Steps   []Step          // Execution steps
    Output  string          // Final result
    Success bool            // Execution outcome
}

type Step struct {
    Type      string        // "tool_call", "llm_response", "error"
    Content   string        // Step details
    Timestamp time.Time
    ToolName  string        // For tool_call steps
    Result    string        // Tool output
}
```

### Learning Paths

**Path A: Quick Generation** (default)

```
Trajectory → formatted summary → Generator.GenerateBullets()
    ↓
LLM prompt: "Extract 3-5 key insights from this execution"
    ↓
Generated bullets → Curator.Curate() → deduplication
    ↓
Delta operations → Playbook update
```

**Path B: Reflection-based** (AutoReflect=true)

```
Trajectory → Reflector.Reflect()
    ↓
Deep analysis: patterns, anti-patterns, edge cases
    ↓
Insights (structured) → Curator.Curate()
    ↓
Quality scoring, deduplication, merge detection
    ↓
High-quality bullets → Delta operations → Playbook
```

### Bullet Structure

```go
type Bullet struct {
    ID           string    // UUID v4
    Content      string    // Knowledge content
    HelpfulCount int       // Positive feedback counter
    HarmfulCount int       // Negative feedback counter
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Source       string    // "trajectory", "manual", "reflection"
}
```

### Deduplication

Curator detects semantic duplicates using cosine similarity:

```
New bullet: "Always validate JWT signatures"
Existing:   "Verify JWT token signatures before use"
    ↓
Similarity: 0.94 (threshold: 0.85)
    ↓
Action: Merge or skip (configurable)
```

### Learning Flow

```
Execution completes with resp.Success
    ↓
TrajectoryContext.ToTrajectory() builds trajectory
    ↓
Check ACEService.config.Generation.AutoReflect
    ↓
    ├─ true:  GenerateBulletsWithReflectionFromTrajectory()
    │          ├─ Reflector.Reflect() analyzes trajectory
    │          └─ Curator.Curate() ensures quality
    │
    └─ false: GenerateBullets() quick extraction
    ↓
learnedBullets (0-N bullets)
    ↓
EventACELearned emitted (if EmitACEEvents=true)
    ↓
TUI displays: "◆ Learned 2 new insights from successful execution:"
              "  • JWT tokens should expire within 15 minutes..."
              "  • Use bcrypt cost factor 12 for passwords..."
    ↓
Playbook persisted to disk (JSON format)
```

## AGENT EXECUTION FLOW

Complete flow integrating tools, ACE retrieval, and learning:

```
1. User submits task: "implement JWT authentication"
    ↓
2. Agent.Execute() starts with TrajectoryContext
    ↓
3. Turn 0: Progressive retrieval fires (TriggerInitial)
    ├─ Query: "implement JWT authentication"
    ├─ Retrieve top 5 bullets from playbook
    ├─ Cache bullets, emit EventACERetrieval
    └─ TUI shows: "⟐ Retrieved 5 new strategies:"
    ↓
4. LLM receives prompt with retrieved bullets:
    User: implement JWT authentication
    
    Retrieved strategies:
    - Use bcrypt for password hashing with cost 12
    - Validate JWT signatures with RS256 algorithm
    - Set token expiration to 15 minutes max
    - Store refresh tokens in httpOnly cookies
    - Never log sensitive authentication data
    ↓
5. LLM plans approach and calls tools:
    ├─ read_file("auth/user.go")
    ├─ write_file("auth/jwt.go", content)
    └─ shell_command("go test ./auth/...")
    ↓
6. Turn 1: Tests fail (error detected)
    ├─ Progressive retrieval fires (TriggerError)
    ├─ Query: "implement JWT authentication" + "test failed" + "signature invalid"
    ├─ Retrieve error-specific bullets
    └─ TUI shows: "⟐ Retrieved 2 new strategies:"
    ↓
7. LLM fixes issue with error-specific guidance
    ├─ apply_patch(fix-signature-validation.patch)
    └─ shell_command("go test ./auth/...")
    ↓
8. Turn 2: Tests pass, LLM responds with completion
    ↓
9. Agent.Execute() completes with Success=true
    ↓
10. Learning pipeline activates:
    ├─ Build trajectory from 8 steps
    ├─ Reflector.Reflect() extracts insights:
    │   - "JWT signature validation requires exact algorithm match"
    │   - "Test coverage for auth edge cases prevents production bugs"
    ├─ Curator.Curate() checks for duplicates
    ├─ Add 2 new bullets to playbook
    └─ TUI shows: "◆ Learned 2 new insights from successful execution:"
    ↓
11. Playbook saved to ~/.spin/ace/playbook.json
```

## TOOL INTEGRATION

ACE bullets guide tool selection and usage:

```go
// Agent loop pseudocode
for turn < maxTurns {
    // 1. Progressive retrieval
    if shouldRetrieve, trigger := shouldRetrieveProgressive(trajCtx); shouldRetrieve {
        query := buildQueryFromContext(trajCtx, trigger)
        bullets := Retrieve(query)
        trajCtx.RecordRetrieval(bullets)
    }
    
    // 2. Build LLM prompt with bullets
    activeBullets := trajCtx.GetActiveBullets()
    prompt := buildPrompt(messages, task, activeBullets)
    
    // 3. LLM responds with tool calls
    response := callLLM(prompt)
    
    // 4. Execute tools
    for _, toolCall := range response.ToolCalls {
        result := executeTool(toolCall)
        trajCtx.RecordStep(toolCall, result)
    }
    
    // 5. Check triggers for next retrieval
}

// 6. After completion: learn from trajectory
if aceEnabled {
    trajectory := trajCtx.ToTrajectory()
    bullets := GenerateBulletsFromTrajectory(trajectory)
    playbook.Add(bullets)
}
```

## CONFIGURATION

### Full Configuration

```yaml
ace:
  enabled: true
  playbook_path: ~/.spin/ace/playbook.json
  trajectory_path: ~/.spin/ace/trajectories/
  
  # Retrieval settings
  top_k: 5                      # Max bullets per query
  min_score: 0.7                # Similarity threshold
  
  # Progressive context
  retrieval:
    progressive_context:
      enabled: true
      cache_ttl: 3              # Bullet lifetime (turns)
      error_lookback: 3         # Steps to check for errors
      tool_change_lookback: 2   # Steps to check for tool changes
      emit_ace_events: true     # Show hints in TUI/exec mode
  
  # Learning configuration
  itemized_learning:
    enabled: true               # Parse feedback from responses
    parse_feedback: true        # Extract helpful/harmful signals
    update_async: true          # Non-blocking bullet updates
  
  generation:
    enabled: true               # Generate bullets from trajectories
    auto_reflect: true          # Use Reflector (vs quick generation)
  
  # Online learning orchestration
  adapter:
    enabled: true
    utility_threshold: 0.1      # Min utility score to keep bullets
    max_memory_size: 1000       # Max bullets in memory
  
  # Playbook refinement
  refine:
    enabled: true
    mode: proactive             # "proactive" or "lazy"
    max_bullets: 1000           # Trigger refinement threshold
    max_tokens: 500000          # Token budget for playbook
    min_utility_score: 0.1      # Prune bullets below this
    check_interval: 100         # Turns between refinement checks
```

### Minimal Configuration

```yaml
ace:
  enabled: true
  # Uses defaults:
  # - playbook_path: ~/.spin/ace/playbook.json
  # - top_k: 5
  # - min_score: 0.7
  # - progressive_context enabled with defaults
  # - auto_reflect: true (deep learning)
```

## VISUAL FEEDBACK

### Retrieval Hints (TUI/Exec Mode)

When progressive retrieval fires:

```
⟐ Retrieved 3 new strategies:
  • Use bcrypt cost factor 12 for password hashing to balance security and performance
  • Validate JWT signatures with RS256 algorithm before trusting token claims
  • Set token expiration to 15 minutes maximum for security-critical operations
```

Symbol: `⟐` (green)  
Format: First line only, max 120 chars per bullet  
Deduplication: Only shows truly new bullets (not seen before in session)

### Learning Hints (TUI/Exec Mode)

After execution completes:

```
◆ Learned 2 new insights from successful execution:
  • JWT signature validation requires exact algorithm match to prevent "alg:none" attacks
  • Comprehensive test coverage for authentication edge cases prevents production security bugs
```

Symbol: `◆` (blue)  
Status indicator: Green "successful" or yellow "failed"  
Deduplication: Only shows new insights (not previously learned)

### Disabling Hints

```yaml
ace:
  retrieval:
    progressive_context:
      emit_ace_events: false    # Suppress all ACE visual feedback
```

## PLAYBOOK FORMAT

Playbook stored as JSON at `~/.spin/ace/playbook.json`:

```json
{
  "bullets": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "content": "Use bcrypt cost factor 12 for password hashing",
      "helpful_count": 5,
      "harmful_count": 0,
      "created_at": "2024-11-07T10:00:00Z",
      "updated_at": "2024-11-07T14:30:00Z",
      "source": "reflection"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "content": "Validate JWT signatures with RS256 algorithm",
      "helpful_count": 3,
      "harmful_count": 0,
      "created_at": "2024-11-07T11:00:00Z",
      "updated_at": "2024-11-07T11:00:00Z",
      "source": "trajectory"
    }
  ],
  "version": "1.0",
  "embeddings": {
    "model": "nomic-embed-text",
    "dimension": 768
  }
}
```

## EMBEDDINGS

ACE uses Ollama for vector embeddings:

**Model**: `nomic-embed-text` (768 dimensions)  
**Similarity**: Cosine similarity (range: 0-1)  
**Fallback**: Mock embedder if Ollama unavailable

Embedding flow:
1. Bullet content → Ollama API
2. Vector stored alongside bullet
3. Query → Ollama API → query vector
4. Cosine similarity: `dot(query_vec, bullet_vec) / (||query|| * ||bullet||)`
5. Top-K bullets with score ≥ min_score returned

## COMPONENTS

### ACEService

Central orchestrator managing all ACE subsystems.

Methods:
- `Retrieve(ctx, query)`: Semantic search for bullets
- `GenerateBullets(ctx, content, source)`: Quick bullet generation
- `GenerateBulletsWithReflectionFromTrajectory(ctx, traj)`: Deep analysis pipeline
- `ApplyFeedback(ctx, feedback)`: Update bullet scores

### Retriever

Semantic search using vector embeddings.

Algorithm:
1. Embed query with Ollama
2. Compute cosine similarity against all bullets
3. Filter by min_score threshold
4. Return top-K ranked by similarity

### Reflector

Deep trajectory analysis for insight extraction.

Prompt structure:
```
Analyze this execution trajectory and extract key insights.

Task: {query}

Steps:
1. [tool_call] read_file("auth.go")
   Result: {content}
2. [llm_response] Planning authentication flow
3. [tool_call] write_file("jwt.go", {content})
   Result: success

Identify:
- Success patterns (what worked well)
- Error patterns (what failed and why)
- Edge cases (boundary conditions handled)
- Best practices (techniques worth reusing)
```

### Curator

Quality control and deduplication.

Operations:
- Semantic duplicate detection (cosine similarity ≥ 0.85)
- Merge similar bullets (combine counters, update content)
- Filter low-quality bullets (utility score < threshold)
- Apply delta operations for atomic updates

### TrajectoryContext

Execution state tracking during agent loop.

State:
```go
type TrajectoryContext struct {
    Query              string
    Steps              []Step
    CurrentTurn        int
    LastRetrievalTurn  int
    BulletCache        map[string]CachedBullet
    RetrievalEvents    []RetrievalEvent
    CacheHits          int
    CacheMisses        int
}
```

Methods:
- `RecordRetrieval(event, bullets)`: Cache retrieved bullets
- `RecordStep(step)`: Track execution step
- `GetActiveBullets()`: Return TTL-filtered bullets
- `HasRecentError(lookback)`: Check for errors in last N steps
- `GetRecentTools(lookback)`: Extract tool names from last N steps
- `ToTrajectory()`: Build final trajectory for learning

## PERFORMANCE

### Retrieval Performance

```
Playbook size: 1000 bullets
Query latency: ~50ms (Ollama embedding + similarity)
Cache hit rate: 60-80% (typical)
Bullets per retrieval: 3-5 (configurable)
```

### Learning Performance

```
Quick generation: ~2s (simple LLM call)
Reflection-based: ~5-10s (Reflector + Curator pipeline)
Playbook save: <100ms (JSON serialization)
```

### Memory Usage

```
Bullet: ~500 bytes (content + metadata + embedding)
1000 bullets: ~500KB in memory
Playbook JSON: ~800KB on disk
```

## EXAMPLES

### Example 1: Authentication Implementation

```bash
$ spin exec "implement JWT authentication with bcrypt password hashing"

# Turn 0: Initial retrieval
⟐ Retrieved 5 new strategies:
  • Use bcrypt cost factor 12 for password hashing
  • Validate JWT signatures with RS256 algorithm
  • Set token expiration to 15 minutes max
  • Store refresh tokens in httpOnly cookies
  • Never log password hashes or raw passwords

# Agent implements with guidance from bullets
# ... tool calls, code writing ...

# Completion
◆ Learned 3 new insights from successful execution:
  • JWT middleware should reject tokens with "alg:none" header
  • Password validation timing should be constant to prevent timing attacks
  • Test coverage for token expiration edge cases prevents production bugs
```

### Example 2: Error Recovery

```bash
$ spin exec "fix failing authentication tests"

# Turn 0: Initial retrieval
⟐ Retrieved 4 new strategies:
  • Run tests with -v flag for detailed output
  • Check error messages for root cause hints
  • Use test fixtures for reproducible failures
  • Verify test environment matches production

# Turn 1: Tests fail with specific error
# Progressive retrieval fires (TriggerError)
⟐ Retrieved 2 new strategies:
  • JWT signature validation requires exact algorithm match
  • Test mocks should use same crypto implementation as production

# Agent applies error-specific guidance
# ... fixes issue, tests pass ...

◆ Learned 1 new insight from successful execution:
  • JWT test fixtures must include valid signature with matching algorithm
```

### Example 3: Disabled ACE

```yaml
# config.yaml
ace:
  enabled: false
```

```bash
$ spin exec "implement user registration"

# No retrieval hints
# No learning hints
# Agent operates without ACE guidance
```

## DEVELOPMENT

### Adding Custom Bullets

Manual bullet injection:

```go
import "github.com/dmytrogajewski/spin/internal/ace/bullet"

b := &bullet.Bullet{
    ID:      uuid.New().String(),
    Content: "Custom best practice for my domain",
    Source:  "manual",
}

aceService.playbook.Add(b)
aceService.playbook.Save()
```

### Testing ACE Pipeline

```bash
# Unit tests
go test ./internal/ace/...

# Integration test with live agent
go test ./internal/agent -run TestACEIntegration

# Playbook inspection
cat ~/.spin/ace/playbook.json | jq '.bullets[] | select(.helpful_count > 5)'
```

### Debugging Retrieval

Enable debug logging:

```bash
export SPIN_DEBUG=1
spin exec "task"
```

Logs show:
```
[DEBUG] ACE retrieval triggered: turn=0, trigger=initial
[DEBUG] Query: "implement authentication"
[DEBUG] Retrieved 5 bullets, scores: [0.92, 0.87, 0.83, 0.79, 0.71]
[DEBUG] Cache size: 5, hits: 0, misses: 1
```

## FILES

```
~/.spin/ace/playbook.json           - Main playbook storage
~/.spin/ace/trajectories/           - Execution trajectories (optional)
~/.spin/ace/embeddings.cache        - Embedding cache (future)
```

## ENVIRONMENT

```
OLLAMA_HOST              - Ollama API endpoint (default: http://localhost:11434)
SPIN_ACE_PLAYBOOK        - Override playbook path
SPIN_ACE_DISABLE         - Disable ACE (equivalent to enabled: false)
```

## ARCHITECTURE PATTERNS

### Service Separation

ACE is isolated from core agent logic:

```
Agent (core execution)
  ↓ (optional dependency)
ACEService (learning & retrieval)
  ↓
Playbook, Retriever, Reflector, Curator (ACE internals)
```

Benefits:
- Agent works without ACE (graceful degradation)
- ACE can be tested independently
- Clear separation of concerns

### Builder Pattern

ACE configuration via Builder:

```go
aceService, err := NewACEServiceBuilder(cfg, workDir).
    WithLLM(provider, modelName).
    WithMaxTokens(4096).
    Build()
```

### Event-Driven UI

ACE emits events for UI consumption:

```go
// Retrieval event
emitter.Emit(Event{
    Type: EventACERetrieval,
    Data: ACERetrievalData{
        Bullets: bullets,
        Trigger: trigger,
    },
})

// Learning event
emitter.Emit(Event{
    Type: EventACELearned,
    Data: ACELearningData{
        Bullets: bullets,
        Success: success,
    },
})
```

UI layer (TUI mapper) handles display without coupling to ACE internals.

## SEE ALSO

spin(1), spin-config(5), spin-tools(7)

## AUTHORS

ACE pipeline designed for autonomous agent improvement through trajectory analysis.

Progressive retrieval implements adaptive context injection based on execution state.
