---
name: journey
description: Journey-based feature requirements with CJM
---

# JOURNEY-<date>: <description>

## Roadmap Link
- Source roadmap: <source_roadmap_link>
- Feature: <feature_name>

## 1. Journey

When **<user persona and context>** I want to **<action or capability>** so I can **<outcome or value delivered>**.

## 2. CJM

<Brief context paragraph: who the user is, what they are trying to do, what friction exists today, and how this feature removes that friction.>

### Phase 1: <phase_name>

**User Intent:** <What the user is trying to accomplish in this phase.>

**Actions:** <Concrete steps the user takes.>

**Pain / Risk:** <What could go wrong, what is confusing, what could fail. At least 3 scenarios per phase.>

**Success Signal:** <Observable evidence that this phase completed correctly.>

### Phase 2: <phase_name>

**User Intent:** <What the user is trying to accomplish in this phase.>

**Actions:** <Concrete steps the user takes.>

**Pain / Risk:** <What could go wrong, what is confusing, what could fail. At least 3 scenarios per phase.>

**Success Signal:** <Observable evidence that this phase completed correctly.>

### Phase 3: <phase_name>

**User Intent:** <What the user is trying to accomplish in this phase.>

**Actions:** <Concrete steps the user takes.>

**Pain / Risk:** <What could go wrong, what is confusing, what could fail. At least 3 scenarios per phase.>

**Success Signal:** <Observable evidence that this phase completed correctly.>

<!-- Add more phases as needed. Most journeys have 3-6 phases. -->

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| <friction_1> | <phase> | <opportunity_1> |
| <friction_2> | <phase> | <opportunity_2> |
| <friction_3> | <phase> | <opportunity_3> |

### North Star Summary

<One paragraph describing the ideal end state when this journey is fully realized. What does success look like from the user's perspective?>

## 3. UX Implementation and Assessment

### Time to First Value
- [ ] <Metric: how fast the user gets value from this feature>
- [ ] <Metric: onboarding speed>

### Onboarding Clarity
- [ ] <Checkpoint: is the feature discoverable?>
- [ ] <Checkpoint: are error messages clear?>

### Production-Ready Defaults
- [ ] <Checkpoint: are defaults safe and useful?>
- [ ] <Checkpoint: does the feature work without configuration?>

### Golden Path Quality
- [ ] <Checkpoint: does the happy path work end-to-end?>
- [ ] <Checkpoint: is the output correct and complete?>

### Error Quality
- [ ] <Checkpoint: do errors name the problem and suggest a fix?>
- [ ] <Checkpoint: are edge cases handled gracefully?>

### Failure Safety
- [ ] <Checkpoint: is the feature recoverable from mistakes?>
- [ ] <Checkpoint: are destructive operations guarded?>

<!-- Add more assessment categories as needed:
     Decision Load, Progressive Complexity, Runtime Transparency,
     Debuggability, Cross-Surface Consistency, Workflow Consistency,
     Change Safety, Experimentation Safety, Interaction Latency,
     Developer Feedback Speed, Team Scale, System Scale,
     Right Behavior by Default, Anti-Bypass Design -->

## 4. Tests

### TC-01: <test_name>

**Given** <precondition>.
**When** <action>.
**Then** <expected outcome>.

### TC-02: <test_name>

**Given** <precondition>.
**When** <action>.
**Then** <expected outcome>.

### TC-03: <test_name>

**Given** <precondition>.
**When** <action>.
**Then** <expected outcome>.

<!-- Add more test cases as needed. Most journeys have 5-15 test cases. -->

## Traceability
- Roadmap item: <link to roadmap item in specs/>
- Implementation files: <to be filled by /implement>
- Test files: <to be filled by /implement>
