# Feature Requirements Document: Reflector Component

**FRD ID:** FRD-20251030-003  
**Feature:** ACE Reflector Component  
**Phase:** 3 - Learning & Reflection  
**Status:** Draft  
**Created:** 2025-10-30  
**Author:** AI Agent (Spin)

---

## Executive Summary

The Reflector component analyzes execution trajectories to extract concrete, actionable insights from successes and failures. It implements the reflection step of the ACE (Agentic Context Engineering) framework, transforming raw execution traces into structured lessons that can be curated into the playbook.

**Key Goal:** Convert trajectory data into high-quality insights that improve agent performance over time.

---

## Background

### Context

ACE's learning loop consists of:
1. **Generate** (✅ Complete) - Execute tasks with context bullets
2. **Reflect** (🔄 This Feature) - Analyze trajectories to extract insights
3. **Curate** (📋 Next) - Synthesize insights into playbook delta

The Reflector bridges generation and curation by transforming execution data into actionable knowledge.

### Problem Statement

Raw trajectories contain valuable information but are unstructured:
- Success/failure patterns are implicit
- Error modes require analysis to understand
- Multiple trajectories may reveal recurring issues
- Insights need to be actionable, not generic

**Challenge:** Extract concrete, specific insights from trajectories while avoiding:
- Brevity bias (overly short, generic advice)
- Hallucination (inventing facts not in trajectory)
- Redundancy (repeating existing playbook knowledge)

---

## Requirements

### Functional Requirements

#### FR-1: Single Trajectory Analysis
**Priority:** MUST  
**Description:** Analyze a single trajectory to extract insights.

**Acceptance Criteria:**
- Accept Trajectory struct as input
- Parse trajectory steps and outcome
- Identify what worked and what failed
- Generate 1-10 concrete insights per trajectory
- Include confidence scores for insights

**Input:**
```go
type Trajectory struct {
    Query            string
    RetrievedBullets []*bullet.Bullet
    Steps            []TrajectoryStep
    Output           string
    Success          bool
    BulletFeedback   *BulletFeedback
    Metadata         TrajectoryMetadata
}
```

**Output:**
```go
type Insight struct {
    Content     string   // Actionable lesson (max 500 chars)
    Source      string   // Trajectory ID
    Confidence  float64  // 0.0 to 1.0
    Category    string   // "success_pattern", "error_mode", "optimization"
    Evidence    []string // Supporting quotes from trajectory
}
```

#### FR-2: Multi-Iteration Refinement
**Priority:** MUST  
**Description:** Refine insights through multiple LLM iterations.

**Acceptance Criteria:**
- Support up to 5 refinement rounds
- Each round improves specificity and actionability
- Track iteration depth in metadata
- Stop early if quality threshold met
- Configurable iteration count

**Refinement Process:**
1. Initial extraction (broad insights)
2. Specificity pass (make concrete)
3. Actionability pass (ensure useful)
4. De-duplication pass (remove redundant)
5. Final validation (quality check)

#### FR-3: Batch Trajectory Analysis
**Priority:** SHOULD  
**Description:** Analyze multiple trajectories together to find patterns.

**Acceptance Criteria:**
- Accept list of trajectories
- Identify cross-trajectory patterns
- Detect recurring errors
- Find common success strategies
- Generate aggregate insights

**Pattern Types:**
- Recurring errors (same failure across trajectories)
- Success strategies (common approaches that work)
- Anti-patterns (repeated mistakes)
- Edge cases (unusual situations)

#### FR-4: Labeled and Unlabeled Support
**Priority:** MUST  
**Description:** Work with both labeled (ground truth) and unlabeled trajectories.

**Labeled Trajectory:**
- Has `Success` field set explicitly
- May have correct output for comparison
- Higher confidence insights

**Unlabeled Trajectory:**
- No explicit success indication
- Infer quality from execution characteristics
- Lower confidence insights

#### FR-5: Prompt Template Management
**Priority:** MUST  
**Description:** Use structured prompts that avoid brevity bias.

**Acceptance Criteria:**
- Template for single trajectory reflection
- Template for batch trajectory reflection
- Templates encourage comprehensive analysis
- Prompts request evidence-based insights
- No prompts that encourage brevity

**Prompt Structure:**
```
Analyze this execution trajectory:
[Trajectory data]

Extract concrete, actionable insights. For each insight:
1. State the specific lesson learned
2. Provide evidence from the trajectory
3. Explain when to apply this lesson
4. Rate confidence (0.0 to 1.0)

Aim for comprehensive coverage, not brevity.
```

#### FR-6: Confidence Scoring
**Priority:** MUST  
**Description:** Score each insight's reliability.

**Confidence Factors:**
- Evidence strength (more steps supporting = higher)
- Trajectory completeness (complete = higher)
- Success/failure clarity (clear outcome = higher)
- Insight specificity (concrete = higher)

**Score Range:** 0.0 (low) to 1.0 (high)

#### FR-7: Quality Validation
**Priority:** SHOULD  
**Description:** Validate insight quality before returning.

**Quality Metrics:**
- Length: 50-500 characters
- Specificity: Contains concrete action verbs
- Actionability: Describes "how" or "when"
- Evidence: Backed by trajectory data
- Novelty: Not duplicating existing playbook

---

### Non-Functional Requirements

#### NFR-1: Performance
- Single trajectory analysis: < 5 seconds
- Batch analysis (10 trajectories): < 30 seconds
- Refinement iteration: < 3 seconds per round

#### NFR-2: Reliability
- Handle incomplete trajectories gracefully
- Never crash on malformed input
- Return partial results on timeout

#### NFR-3: Testability
- Unit test coverage ≥ 90%
- Integration tests with real trajectories
- Mock LLM for deterministic testing

#### NFR-4: Observability
- Log each reflection stage
- Emit events for monitoring
- Track iteration counts and confidence

---

## Architecture

### Package Structure

```
internal/ace/reflector/
├── reflector.go        # Main Reflector service
├── insight.go          # Insight data structures
├── prompt.go           # Reflection prompt templates
├── validator.go        # Insight quality validation
├── reflector_test.go   # Unit tests
└── integration_test.go # Integration tests
```

### Data Structures

#### Insight

```go
package reflector

type Insight struct {
    // Content is the actionable lesson
    Content string
    
    // Source is the trajectory ID
    Source string
    
    // Confidence is reliability score (0.0 to 1.0)
    Confidence float64
    
    // Category classifies the insight type
    Category InsightCategory
    
    // Evidence are supporting quotes from trajectory
    Evidence []string
    
    // Iteration is refinement round when created
    Iteration int
    
    // CreatedAt is timestamp
    CreatedAt time.Time
}

type InsightCategory string

const (
    CategorySuccessPattern InsightCategory = "success_pattern"
    CategoryErrorMode      InsightCategory = "error_mode"
    CategoryOptimization   InsightCategory = "optimization"
    CategoryAntiPattern    InsightCategory = "anti_pattern"
)
```

#### ReflectionRequest

```go
type ReflectionRequest struct {
    // Trajectories to analyze
    Trajectories []*generator.Trajectory
    
    // MaxIterations for refinement (default 3)
    MaxIterations int
    
    // MinConfidence threshold (default 0.5)
    MinConfidence float64
    
    // Model to use (default from config)
    Model string
    
    // Temperature for LLM (default 0.3 for consistency)
    Temperature float64
}
```

#### ReflectionResponse

```go
type ReflectionResponse struct {
    // Insights extracted
    Insights []*Insight
    
    // Iterations performed
    Iterations int
    
    // TotalTokens used
    TotalTokens int
    
    // Duration of reflection
    Duration time.Duration
}
```

### Core Interface

```go
type Reflector interface {
    // Reflect analyzes trajectories and extracts insights
    Reflect(ctx context.Context, req ReflectionRequest) (*ReflectionResponse, error)
    
    // RefineInsights improves existing insights
    RefineInsights(ctx context.Context, insights []*Insight, iterations int) ([]*Insight, error)
    
    // ValidateInsight checks insight quality
    ValidateInsight(insight *Insight) error
}
```

### Implementation

```go
type reflector struct {
    llm              llm.Provider
    promptBuilder    *PromptBuilder
    validator        *InsightValidator
    maxIterations    int
    minConfidence    float64
    defaultModel     string
    defaultTemp      float64
}

func NewReflector(llm llm.Provider, opts ...Option) Reflector {
    r := &reflector{
        llm:           llm,
        promptBuilder: NewPromptBuilder(),
        validator:     NewInsightValidator(),
        maxIterations: 3,
        minConfidence: 0.5,
        defaultModel:  "gpt-4",
        defaultTemp:   0.3,
    }
    
    for _, opt := range opts {
        opt(r)
    }
    
    return r
}
```

---

## Implementation Plan

### Phase 1: Core Data Structures (Week 1, Days 1-2)
**Tasks:**
1. Create `internal/ace/reflector` package
2. Define `Insight` struct with validation
3. Define `ReflectionRequest` and `ReflectionResponse`
4. Implement `InsightCategory` enum

**Tests:**
- Insight validation (length, content)
- Category enum values
- Request/response construction

**DoD:**
- All types compile
- Validation tests pass
- 100% coverage on data structures

### Phase 2: Prompt Templates (Week 1, Days 3-4)
**Tasks:**
1. Create `PromptBuilder` type
2. Implement single trajectory prompt template
3. Implement batch trajectory prompt template
4. Add evidence extraction logic

**Tests:**
- Template rendering with sample data
- Evidence extraction from trajectories
- Prompt format validation

**DoD:**
- Prompts render correctly
- Evidence extraction works
- 90% coverage

### Phase 3: Insight Validator (Week 1, Day 5)
**Tasks:**
1. Create `InsightValidator` type
2. Implement length validation (50-500 chars)
3. Implement specificity check (action verbs)
4. Implement confidence validation (0.0-1.0)

**Tests:**
- Valid insights pass
- Invalid insights fail with specific errors
- Edge cases (empty, too long, etc.)

**DoD:**
- All validation rules work
- Clear error messages
- 95% coverage

### Phase 4: Single Trajectory Reflection (Week 2, Days 1-3)
**Tasks:**
1. Implement `Reflect()` method for single trajectory
2. LLM integration for insight extraction
3. Parse LLM response into Insight structs
4. Confidence scoring logic

**Tests:**
- Mock LLM with known responses
- Success and failure trajectories
- Incomplete trajectory handling
- Confidence calculation

**DoD:**
- Single trajectory reflection works
- Insights extracted correctly
- 90% coverage

### Phase 5: Multi-Iteration Refinement (Week 2, Days 4-5)
**Tasks:**
1. Implement `RefineInsights()` method
2. Iterative refinement loop
3. Quality improvement tracking
4. Early stopping logic

**Tests:**
- Refinement improves quality
- Early stopping when threshold met
- Max iterations respected
- Iteration tracking

**DoD:**
- Refinement loop works
- Quality improves per iteration
- 90% coverage

### Phase 6: Batch Analysis (Week 3, Days 1-2)
**Tasks:**
1. Extend `Reflect()` for multiple trajectories
2. Cross-trajectory pattern detection
3. Aggregate insight generation
4. De-duplication logic

**Tests:**
- Batch with 2, 5, 10 trajectories
- Pattern detection across trajectories
- Duplicate removal

**DoD:**
- Batch analysis works
- Patterns detected
- 85% coverage

### Phase 7: Integration & Polish (Week 3, Days 3-5)
**Tasks:**
1. Integration tests with real Generator output
2. Performance benchmarking
3. Documentation
4. Examples

**Tests:**
- End-to-end with Generator
- Performance under target
- Edge cases

**DoD:**
- All integration tests pass
- Performance targets met
- Documentation complete

---

## Testing Strategy

### Unit Tests

**Coverage Target:** ≥ 90%

**Test Categories:**
1. Data structure validation
2. Prompt template rendering
3. Insight parsing from LLM responses
4. Confidence scoring
5. Quality validation

**Key Test Cases:**
```go
func TestReflector_SingleTrajectory_Success(t *testing.T)
func TestReflector_SingleTrajectory_Failure(t *testing.T)
func TestReflector_IncompleteTrajectory(t *testing.T)
func TestReflector_MultiIteration(t *testing.T)
func TestReflector_BatchAnalysis(t *testing.T)
func TestInsightValidator_ValidInsight(t *testing.T)
func TestInsightValidator_InvalidLength(t *testing.T)
func TestInsightValidator_InvalidConfidence(t *testing.T)
```

### Integration Tests

**Tests:**
1. Reflector + Generator integration
2. Real trajectory from coding task
3. Batch reflection on 10 trajectories
4. Refinement quality improvement

### Table-Driven Tests

All validation and parsing logic should use table-driven tests:

```go
func TestInsightValidation(t *testing.T) {
    tests := []struct {
        name    string
        insight *Insight
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid insight",
            insight: &Insight{
                Content: "Always validate input parameters",
                Confidence: 0.8,
            },
            wantErr: false,
        },
        {
            name: "content too short",
            insight: &Insight{
                Content: "short",
                Confidence: 0.8,
            },
            wantErr: true,
            errMsg: "content length",
        },
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateInsight(tt.insight)
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

---

## Success Criteria

### Functional Success
- ✅ Extract insights from single trajectory
- ✅ Refine insights through multiple iterations
- ✅ Analyze batches of trajectories
- ✅ Confidence scoring works
- ✅ Quality validation enforced

### Quality Success
- ✅ Test coverage ≥ 90%
- ✅ No lint errors
- ✅ Race detector clean
- ✅ All integration tests pass

### Performance Success
- ✅ Single trajectory: < 5s
- ✅ Batch (10): < 30s
- ✅ Refinement iteration: < 3s

---

## Dependencies

### Internal Dependencies
- ✅ `internal/ace/generator` (Trajectory structures)
- ✅ `internal/ace/bullet` (Bullet structures)
- ✅ `internal/llm` (LLM provider interface)

### External Dependencies
- OpenAI SDK (for LLM calls)
- None (self-contained)

---

## Risks and Mitigations

### Risk 1: LLM Hallucination
**Impact:** HIGH  
**Probability:** MEDIUM  
**Mitigation:**
- Require evidence from trajectory
- Validate insights against trajectory data
- Use temperature 0.3 for consistency
- Multiple refinement passes

### Risk 2: Brevity Bias
**Impact:** HIGH  
**Probability:** MEDIUM  
**Mitigation:**
- Explicit prompts requesting comprehensive insights
- Validate minimum length (50 chars)
- Reward specificity in prompts
- Multi-iteration refinement

### Risk 3: Poor Quality Insights
**Impact:** MEDIUM  
**Probability:** MEDIUM  
**Mitigation:**
- Quality validator with clear rules
- Confidence scoring
- Min confidence threshold
- Human review option (Phase 4+)

### Risk 4: Performance Issues
**Impact:** LOW  
**Probability:** LOW  
**Mitigation:**
- Async processing option
- Batch size limits
- Timeout handling
- Progress events

---

## Future Enhancements (Post-MVP)

### Phase 4+
- Clustering similar insights
- Insight merging/deduplication
- User feedback on insight quality
- Automated insight categorization
- Cross-playbook insight transfer
- TUI visualization of insights

---

## Appendix A: Example Prompts

### Single Trajectory Reflection Prompt

```
You are analyzing an execution trajectory to extract actionable coding insights.

# Trajectory
Query: {{.Query}}
Success: {{.Success}}
Steps: {{.Steps}}
Output: {{.Output}}

# Task
Extract 3-10 concrete, actionable insights from this trajectory.

For each insight, provide:
1. Content: A specific lesson learned (50-500 chars)
2. Evidence: Quote from trajectory supporting this
3. Confidence: Your confidence in this insight (0.0 to 1.0)
4. Category: success_pattern | error_mode | optimization | anti_pattern

Format as JSON array:
[
  {
    "content": "...",
    "evidence": ["..."],
    "confidence": 0.85,
    "category": "success_pattern"
  }
]

Prioritize:
- Specificity over generality
- Actionability over description
- Evidence over intuition
- Comprehensive coverage over brevity
```

### Batch Trajectory Reflection Prompt

```
You are analyzing multiple execution trajectories to find patterns.

# Trajectories
{{range .Trajectories}}
ID: {{.ID}}
Query: {{.Query}}
Success: {{.Success}}
Output: {{.Output}}
{{end}}

# Task
Identify cross-trajectory patterns and insights.

Look for:
- Recurring errors across trajectories
- Common success strategies
- Anti-patterns (repeated mistakes)
- Edge cases

Format as JSON array of insights (same structure as single trajectory).
```

---

## Appendix B: Validation Rules

### Insight Content Validation

| Rule | Constraint | Error Message |
|------|-----------|---------------|
| Min length | ≥ 50 chars | "insight content too short (min 50)" |
| Max length | ≤ 500 chars | "insight content too long (max 500)" |
| Not empty | len > 0 | "insight content cannot be empty" |
| Action verb | Contains verb | "insight should be actionable" |

### Confidence Validation

| Rule | Constraint | Error Message |
|------|-----------|---------------|
| Min value | ≥ 0.0 | "confidence cannot be negative" |
| Max value | ≤ 1.0 | "confidence cannot exceed 1.0" |

### Category Validation

| Rule | Constraint | Error Message |
|------|-----------|---------------|
| Valid enum | In enum list | "invalid insight category" |

---

**Document Status:** Ready for Implementation  
**Next Steps:** Begin Phase 1 - Core Data Structures  
**Estimated Completion:** 3 weeks from start
