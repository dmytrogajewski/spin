# BUG-20251119: Plan Detection Too Aggressive in ACP Protocol

**Status**: Fixed  
**Priority**: High  
**Component**: `internal/protocol/acp`  
**Reported**: 2025-11-19  
**Fixed**: 2025-11-19

## Summary

The `detectPlanFromOutput` function in the ACP protocol adapter was incorrectly treating **every line** of LLM output as a plan entry once it detected any numbered list or bullet point. This resulted in the UI showing 83+ "plan tasks" when the LLM was providing a comprehensive project review with multiple sections, code examples, and explanations.

## Root Cause

The bug was in `/internal/protocol/acp/notifications.go`, specifically in the `detectPlanFromOutput` function:

1. **Infinite Plan Section**: Once `inPlanSection = true` was set (upon detecting any numbered list or bullet), the flag was **never reset to `false`**
2. **Every Subsequent Line Became a Plan Entry**: After the first numbered list, every remaining line in the LLM output was treated as a potential plan entry
3. **No Exit Condition**: There was no logic to exit the plan section after encountering non-plan content

### Original Problematic Logic

```go
// OLD CODE - BUGGY
for _, line := range lines {
    // ...
    if matchesPlanPattern(line) {
        inPlanSection = true  // ← Set to true
    }
    
    if !inPlanSection {
        continue
    }
    
    // Extract plan entry - THIS RUNS FOR ALL REMAINING LINES!
    // No way to exit plan section
    entry := extractPlanEntry(line, currentPlanPrefix)
    if entry != nil {
        entries = append(entries, *entry)
    }
}
```

## Impact

**User-Visible Symptoms:**
- Plan section showing 83+ tasks when asking LLM to "review this project"
- Every sentence, code line, and explanation becoming a "plan entry"
- Cluttered UI with irrelevant "tasks" that aren't actually tasks
- Confusing UX where the plan section becomes unusable

**Example Scenario:**
```
User: "review this project"

LLM Response:
I'll help you review this project.

## Project Structure

1. Model Architecture     ← PLAN STARTS HERE
2. Data Pipeline
3. Inference System

### 1. Model Architecture    ← ALSO TREATED AS PLAN
Uses Llama-3.2-1B...        ← ALSO TREATED AS PLAN

### 2. Data Pipeline         ← ALSO TREATED AS PLAN
Point cloud data...         ← ALSO TREATED AS PLAN

Let me analyze...           ← ALSO TREATED AS PLAN
```

Every line after "1. Model Architecture" was treated as a plan entry, resulting in 83+ "tasks".

## Fix

### Approach: Conservative Explicit Header Detection

The fix takes a **conservative approach** that only detects plans after **explicit headers**:

- **Explicit Headers Recognized**: `Plan:`, `Steps:`, `Tasks:`, `## Plan`, `## Steps`
- **Exit Conditions**: 2 consecutive non-plan lines OR 2+ empty lines
- **No Implicit Detection**: Standalone numbered lists (without headers) are NOT treated as plans

### New Logic

```go
// NEW CODE - FIXED
var inPlanSection bool
var consecutiveNonPlanLines int

for _, line := range lines {
    // ONLY enter plan section after explicit headers
    if strings.HasPrefix(lowerLine, "plan:") || 
       strings.HasPrefix(lowerLine, "steps:") ||
       strings.HasPrefix(lowerLine, "tasks:") {
        inPlanSection = true
        consecutiveNonPlanLines = 0
        continue
    }

    // Skip if not in explicitly declared plan section
    if !inPlanSection {
        continue
    }

    isPlanPattern := matchesPlanPattern(line)

    // Exit plan section after 2 consecutive non-plan lines
    if !isPlanPattern {
        consecutiveNonPlanLines++
        if consecutiveNonPlanLines >= 2 {
            inPlanSection = false  // ← EXIT PLAN SECTION
            consecutiveNonPlanLines = 0
        }
        continue
    }

    consecutiveNonPlanLines = 0
    
    entry := extractPlanEntry(line, "")
    if entry != nil {
        entries = append(entries, *entry)
    }
}
```

### Design Trade-offs

**Conservative Approach Benefits:**
- ✅ Avoids false positives (code examples, enumerations, explanations)
- ✅ Predictable behavior (only explicit plan sections)
- ✅ Better UX (plan section shows actual plans)

**Conservative Approach Limitations:**
- ⚠️ Misses implicit plans without headers (e.g., standalone "1. 2. 3." lists)
- ⚠️ Requires LLM to use explicit headers

**Rationale:** Implicit numbered lists appear in many non-plan contexts (code line numbers, explanations, feature lists, etc.). The false positive rate is too high. It's better to miss some implicit plans than to pollute the UI with 83 fake "tasks".

## Test Coverage

### New Tests Added

1. **`internal/protocol/acp/plan_detection_fix_test.go`**
   - Tests plan section exit conditions
   - Tests code examples aren't treated as plans
   - Tests multiple disconnected plan sections
   - Tests real-world LLM response scenario

2. **`internal/protocol/acp/plan_unit_test.go`**
   - Unit tests for `detectPlanFromOutput`
   - Unit tests for `matchesPlanPattern`
   - Unit tests for `extractPlanEntry`
   - Tests priority detection

### Test Results

All tests pass:

```bash
$ go test ./internal/protocol/acp/plan_detection_fix_test.go ...
=== RUN   TestDetectPlanFromOutput_ExitsPlanSection
=== RUN   TestDetectPlanFromOutput_RealWorldScenario
--- PASS: TestDetectPlanFromOutput_ExitsPlanSection (0.00s)
--- PASS: TestDetectPlanFromOutput_RealWorldScenario (0.00s)
PASS
```

## Files Changed

- `internal/protocol/acp/notifications.go` - Fixed `detectPlanFromOutput` logic
- `internal/protocol/acp/plan_detection_fix_test.go` - New comprehensive E2E tests
- `internal/protocol/acp/plan_unit_test.go` - New unit tests

## Verification

### Manual Testing

```bash
# Build with fix
go build -o bin/spin ./cmd/spin

# Test with real LLM
./bin/spin tui
> review this project
```

**Expected Result:**
- Plan section shows ONLY explicit plan items (after "Plan:" or "Steps:" headers)
- Code examples, explanations, and enumerations are NOT shown as plan entries
- Reasonable number of plan entries (3-10), not 83+

## Future Improvements

### Option 1: LLM System Prompt Guidance

Add explicit guidance to the system prompt:

```text
When creating a plan, always use an explicit header:
  Plan:
  1. First step
  2. Second step
```

### Option 2: Structured Plan Detection

Integrate with the full planning system (`internal/planning`) instead of text-based detection:
- Agent creates structured `planning.Plan` objects
- ACP adapter converts `planning.Plan` → `acp.PlanEntry[]`
- No text parsing needed

This is already partially implemented (see `convertOrchestrationPlanToACP` in `plan_converter.go`) but requires the agent to use the planning service.

### Option 3: ML-based Classification

Use an ML model to classify sections of text as "plan" vs "non-plan". This would allow detecting implicit plans without false positives. However, this adds complexity and latency.

## Acceptance Criteria

- [x] `detectPlanFromOutput` properly exits plan sections
- [x] Code examples are not treated as plan entries
- [x] Only explicit "Plan:" / "Steps:" sections are detected
- [x] Comprehensive test coverage for edge cases
- [x] All existing tests pass
- [x] Binary builds successfully
- [ ] Manual verification with real LLM (user to verify)

## References

- User report: Screenshot showing "83 Tasks" in Plan section
- ACP Protocol Spec: `vendordocs/docs/protocol.md`
- Feature 4.2: `specs/frds/FRD-20251114223636-acp-plans-commands.md`
- Planning Service: `internal/planning/service.go`


