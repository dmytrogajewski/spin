# Robust UI Architecture Design

## Core Principles

### 1. Separation of Concerns
- **Output Manager**: Handles transcript scrolling and content display
- **Status Manager**: Manages status bar data and updates
- **Prompt Manager**: Handles input and prompt rendering
- **Layout Manager**: Coordinates positioning without complex ANSI

### 2. Incremental Integration
- Each component can be tested independently
- Components integrate through well-defined interfaces
- No circular dependencies or callback chains

### 3. Failure Isolation
- If one component fails, others continue working
- Clear error boundaries and fallback behavior
- No single point of failure

## Component Architecture

```
┌─────────────────────────────────────────┐
│                UI Controller            │
│  (PureTTY - orchestrates components)    │
└─────────────────┬───────────────────────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
    ▼             ▼             ▼
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Output  │ │ Status  │ │ Prompt  │
│Manager  │ │Manager  │ │Manager  │
└─────────┘ └─────────┘ └─────────┘
```

### Output Manager
- **Responsibility**: Transcript scrolling, content display
- **Interface**: `PrintLine()`, `PrintChunks()`
- **Implementation**: Existing `CoordinatedWriter` (proven stable)

### Status Manager  
- **Responsibility**: Status data, metrics, updates
- **Interface**: `SetStatus()`, `GetStatus()`, `UpdateMetrics()`
- **Implementation**: Simple state container with update notifications

### Prompt Manager
- **Responsibility**: Input handling, prompt rendering
- **Interface**: `Render()`, `HandleInput()`
- **Implementation**: Existing prompt system (proven stable)

### Layout Manager
- **Responsibility**: Coordinate positioning, handle resize
- **Interface**: `Layout()`, `OnResize()`
- **Implementation**: Simple positioning logic without complex ANSI

## Integration Strategy

### Phase 1: Status Data (No UI)
- Create `StatusManager` with metrics storage
- Integrate with existing event system
- Add to `PureTTY` without rendering

### Phase 2: Simple Status Display
- Add status line above prompt (not sticky)
- Use existing `CoordinatedWriter` for rendering
- Test thoroughly before making sticky

### Phase 3: Sticky Positioning (If Needed)
- Only after Phase 2 is proven stable
- Use simple newline-based approach, not ANSI positioning
- Gradual rollout with fallback

## Key Differences from Previous Approach

### ❌ Previous (Failed)
- Single complex component (StickyCoordinator)
- Direct ANSI cursor manipulation
- Complex callback chains
- All-or-nothing integration

### ✅ New (Robust)
- Multiple simple components
- Leverage existing stable systems
- Clear interfaces and boundaries
- Incremental integration with fallbacks

## Testing Strategy

### Component-Level Testing
- Each manager tested independently
- Mock interfaces for integration tests
- No complex PTY testing required initially

### Integration Testing
- Test components together with simple scenarios
- Use existing `CoordinatedWriter` tests as model
- Gradual complexity increase

### Fallback Testing
- Test behavior when components fail
- Ensure graceful degradation
- No deadlocks or unresponsive states

## Implementation Plan

### Step 1: StatusManager (1-2 hours)
- Create simple status data structure
- Add to existing event system
- Unit tests only

### Step 2: Status Display (2-3 hours)  
- Add status line above prompt
- Use existing rendering system
- Integration tests

### Step 3: Sticky (Optional, 3-4 hours)
- Only if Step 2 is stable
- Simple newline-based approach
- Extensive testing

This approach prioritizes stability over complexity and allows for incremental validation at each step.
