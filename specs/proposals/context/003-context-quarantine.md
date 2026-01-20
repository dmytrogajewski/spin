# Proposal: Context Quarantine via Agent Skills

**ID**: PROP-CONTEXT-003  
**Title**: Context Quarantine Through Agent Skills Standard  
**Status**: Draft  
**Created**: 2025-01-20  
**Updated**: 2025-01-20  
**Author**: AI Assistant  
**References**: 
- [LangChain Context Engineering](https://github.com/langchain-ai/how_to_fix_your_context)
- [Agent Skills Specification](https://agentskills.io)
- [Agent Skills GitHub](https://github.com/agentskills/agentskills)

## Summary

Implement context quarantine by adopting the **Agent Skills** open standard - a lightweight, portable format for extending AI agent capabilities with specialized knowledge and procedural instructions. Skills provide natural context isolation through on-demand loading of domain-specific instructions.

## Why Agent Skills?

Agent Skills is an open standard originally developed by Anthropic and now adopted by major AI tools including Claude Code, Cursor, Gemini CLI, VS Code, and others. Using this standard instead of a custom sub-agent architecture provides:

1. **Interoperability**: Skills work across different agent platforms
2. **Ecosystem**: Access to existing skills from the community
3. **Proven Design**: Progressive disclosure minimizes context usage
4. **Simplicity**: Folder-based structure, no complex orchestration
5. **Extensibility**: Users can create custom skills for their workflows

## Problem Statement

### Current State

Spin operates as a single agent with a unified context window:
- All tool results accumulate in one conversation history
- Verbose outputs (e.g., large file contents, command outputs) persist
- No isolation between different task domains
- Context compression is reactive, not proactive

### Identified Issues

1. **Context Clash**: Information from different domains may conflict
2. **Context Pollution**: Large tool outputs pollute context for subsequent operations
3. **No Specialization**: Same context strategy for all task types
4. **Accumulated Noise**: Old tool results remain even when irrelevant

### Context Rot Risks

- **Context Clash**: Research results may conflict with code context
- **Context Confusion**: Mixed domains degrade reasoning quality
- **Context Poisoning**: Errors in one domain propagate to others

## Proposed Solution

### 1. Agent Skills Format

Adopt the SKILL.md specification for defining specialized capabilities.

```
skills/
├── code-review/
│   ├── SKILL.md
│   ├── scripts/
│   │   └── analyze.py
│   └── references/
│       └── CHECKLIST.md
├── debugging/
│   ├── SKILL.md
│   └── references/
│       └── PATTERNS.md
├── testing/
│   ├── SKILL.md
│   └── scripts/
│       └── generate-tests.py
└── git-workflow/
    ├── SKILL.md
    └── references/
        └── CONVENTIONS.md
```

### 2. SKILL.md Format

Each skill follows the Agent Skills specification:

```yaml
---
name: code-review
description: Reviews code for quality, security, and best practices. Use when asked to review, analyze, or audit code. Identifies bugs, security vulnerabilities, performance issues, and style violations.
license: Apache-2.0
compatibility: Designed for Spin agent
metadata:
  author: spin
  version: "1.0"
allowed-tools: Read Grep Glob
---

# Code Review Skill

## Overview
This skill enables thorough code review with focus on:
- Security vulnerabilities
- Performance bottlenecks
- Code quality and maintainability
- Best practices adherence

## Review Process

1. **Understand Context**: Read related files to understand the codebase
2. **Identify Scope**: Determine what files/functions to review
3. **Analyze**: Check against review criteria
4. **Report**: Provide actionable findings

## Review Criteria

### Security
- Input validation
- SQL injection risks
- XSS vulnerabilities
- Authentication/authorization issues
- Sensitive data exposure

### Performance
- N+1 queries
- Unnecessary allocations
- Missing indexes
- Blocking operations

### Quality
- Error handling
- Code duplication
- Function complexity
- Test coverage

## Output Format

Provide findings as:
```
## Summary
[Brief overview]

## Critical Issues
- [Issue with file:line reference]

## Recommendations
- [Suggestion with rationale]
```

See [CHECKLIST.md](references/CHECKLIST.md) for detailed criteria.
```

### 3. Skill Discovery and Loading

Implement the Agent Skills discovery mechanism.

```go
// internal/skills/discovery.go

type SkillDiscovery struct {
    paths     []string
    validator *SkillValidator
}

type SkillMetadata struct {
    Name         string            `yaml:"name"`
    Description  string            `yaml:"description"`
    License      string            `yaml:"license,omitempty"`
    Compatibility string           `yaml:"compatibility,omitempty"`
    Metadata     map[string]string `yaml:"metadata,omitempty"`
    AllowedTools string            `yaml:"allowed-tools,omitempty"`
    Path         string            `yaml:"-"`
}

func (d *SkillDiscovery) Discover() ([]SkillMetadata, error) {
    skills := []SkillMetadata{}
    
    for _, basePath := range d.paths {
        entries, err := os.ReadDir(basePath)
        if err != nil {
            continue
        }
        
        for _, entry := range entries {
            if !entry.IsDir() {
                continue
            }
            
            skillPath := filepath.Join(basePath, entry.Name(), "SKILL.md")
            if _, err := os.Stat(skillPath); os.IsNotExist(err) {
                continue
            }
            
            meta, err := d.parseMetadata(skillPath)
            if err != nil {
                continue
            }
            
            // Validate name matches directory
            if meta.Name != entry.Name() {
                continue
            }
            
            meta.Path = filepath.Join(basePath, entry.Name())
            skills = append(skills, meta)
        }
    }
    
    return skills, nil
}

func (d *SkillDiscovery) parseMetadata(path string) (SkillMetadata, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return SkillMetadata{}, err
    }
    
    // Extract YAML frontmatter
    parts := strings.SplitN(string(content), "---", 3)
    if len(parts) < 3 {
        return SkillMetadata{}, errors.New("invalid SKILL.md format")
    }
    
    var meta SkillMetadata
    if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
        return SkillMetadata{}, err
    }
    
    return meta, nil
}
```

### 4. Context Injection

Inject skill metadata into system prompt using XML format (recommended for Claude).

```go
// internal/skills/injector.go

type SkillInjector struct {
    discovery *SkillDiscovery
}

func (i *SkillInjector) GenerateAvailableSkillsXML() (string, error) {
    skills, err := i.discovery.Discover()
    if err != nil {
        return "", err
    }
    
    var sb strings.Builder
    sb.WriteString("<available_skills>\n")
    
    for _, skill := range skills {
        sb.WriteString("  <skill>\n")
        sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", xmlEscape(skill.Name)))
        sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", xmlEscape(skill.Description)))
        sb.WriteString(fmt.Sprintf("    <location>%s/SKILL.md</location>\n", skill.Path))
        sb.WriteString("  </skill>\n")
    }
    
    sb.WriteString("</available_skills>")
    return sb.String(), nil
}

// Tokens used: ~50-100 per skill (metadata only)
```

### 5. Skill Activation

Load full skill instructions on-demand when activated.

```go
// internal/skills/loader.go

type SkillLoader struct {
    cache map[string]*LoadedSkill
    mu    sync.RWMutex
}

type LoadedSkill struct {
    Metadata     SkillMetadata
    Instructions string           // Full SKILL.md body
    References   map[string]string // Loaded reference files
    Scripts      map[string]string // Available scripts
    LoadedAt     time.Time
}

func (l *SkillLoader) Load(skillPath string) (*LoadedSkill, error) {
    l.mu.RLock()
    if cached, ok := l.cache[skillPath]; ok {
        l.mu.RUnlock()
        return cached, nil
    }
    l.mu.RUnlock()
    
    // Read SKILL.md
    skillFile := filepath.Join(skillPath, "SKILL.md")
    content, err := os.ReadFile(skillFile)
    if err != nil {
        return nil, err
    }
    
    // Parse frontmatter and body
    meta, body := l.parseSkillFile(content)
    
    // Discover available resources (don't load yet - progressive disclosure)
    references := l.discoverReferences(skillPath)
    scripts := l.discoverScripts(skillPath)
    
    loaded := &LoadedSkill{
        Metadata:     meta,
        Instructions: body,
        References:   references,
        Scripts:      scripts,
        LoadedAt:     time.Now(),
    }
    
    l.mu.Lock()
    l.cache[skillPath] = loaded
    l.mu.Unlock()
    
    return loaded, nil
}

func (l *SkillLoader) LoadReference(skillPath, refPath string) (string, error) {
    fullPath := filepath.Join(skillPath, refPath)
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return "", err
    }
    return string(content), nil
}
```

### 6. Skill Activation Tool

Provide a tool for the agent to activate skills.

```go
// internal/tools/skill_tool.go

type SkillTool struct {
    loader    *SkillLoader
    discovery *SkillDiscovery
}

func (t *SkillTool) Definition() llm.Tool {
    return llm.Tool{
        Name:        "skill",
        Description: "Activate a specialized skill to get detailed instructions for a task domain. Use when you need domain-specific guidance.",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "action": {
                    "type": "string",
                    "enum": []string{"list", "activate", "reference"},
                    "description": "Action to perform",
                },
                "name": {
                    "type": "string",
                    "description": "Skill name (for activate/reference)",
                },
                "reference": {
                    "type": "string", 
                    "description": "Reference file path within skill (for reference action)",
                },
            },
            "required": []string{"action"},
        },
    }
}

func (t *SkillTool) Execute(ctx context.Context, params SkillParams) (*ToolResult, error) {
    switch params.Action {
    case "list":
        skills, _ := t.discovery.Discover()
        output := formatSkillList(skills)
        return &ToolResult{Output: output}, nil
        
    case "activate":
        skill, err := t.loader.Load(params.Name)
        if err != nil {
            return &ToolResult{Error: fmt.Sprintf("Skill '%s' not found", params.Name)}, nil
        }
        
        // Return instructions (body of SKILL.md)
        output := fmt.Sprintf("# %s Skill Activated\n\n%s", skill.Metadata.Name, skill.Instructions)
        
        // List available references
        if len(skill.References) > 0 {
            output += "\n\n## Available References\n"
            for ref := range skill.References {
                output += fmt.Sprintf("- %s\n", ref)
            }
            output += "\nUse `skill(action='reference', name='%s', reference='<path>')` to load.\n"
        }
        
        return &ToolResult{Output: output}, nil
        
    case "reference":
        content, err := t.loader.LoadReference(params.Name, params.Reference)
        if err != nil {
            return &ToolResult{Error: fmt.Sprintf("Reference not found: %s", params.Reference)}, nil
        }
        return &ToolResult{Output: content}, nil
    }
    
    return &ToolResult{Error: "Unknown action"}, nil
}
```

### 7. Progressive Disclosure

Skills use progressive disclosure to minimize context usage:

```
Level 1: Metadata (~50-100 tokens per skill)
├── name + description loaded at startup
├── Enables agent to know WHAT skills are available
└── Injected into system prompt

Level 2: Instructions (<5000 tokens recommended)
├── Full SKILL.md body loaded on activation
├── Loaded ONLY when skill is activated
└── Contains detailed step-by-step guidance

Level 3: Resources (as needed)
├── references/ - detailed documentation
├── scripts/ - executable code
├── assets/ - templates, data
└── Loaded on-demand by agent
```

### 8. Built-in Skills

Ship Spin with default skills for common tasks.

```
~/.spin/skills/
├── code-review/
│   └── SKILL.md          # Security, quality, performance review
├── debugging/
│   └── SKILL.md          # Systematic debugging approach
├── testing/
│   └── SKILL.md          # Test generation and coverage
├── refactoring/
│   └── SKILL.md          # Safe refactoring patterns
├── git-workflow/
│   └── SKILL.md          # Commit messages, branching, PRs
├── documentation/
│   └── SKILL.md          # Writing docs, READMEs, comments
└── security-audit/
    └── SKILL.md          # OWASP, vulnerability scanning
```

### 9. Custom Skills

Users can add custom skills:

```yaml
# ~/.spin/skills/my-company-api/SKILL.md
---
name: my-company-api
description: Internal API integration patterns for MyCompany services. Use when working with our REST APIs, authentication, or service mesh.
metadata:
  author: engineering-team
  version: "2.1"
---

# MyCompany API Integration

## Authentication
All APIs use OAuth2 with JWT tokens...

## Services
- user-service: /api/v2/users
- order-service: /api/v1/orders
...
```

### 10. Skill Matching

Match user queries to relevant skills.

```go
// internal/skills/matcher.go

type SkillMatcher struct {
    skills   []SkillMetadata
    embedder embedder.Embedder
    index    *hnsw.Index
}

func (m *SkillMatcher) Match(query string, topK int) ([]MatchedSkill, error) {
    // Embed query
    queryEmb, err := m.embedder.Embed(context.Background(), query)
    if err != nil {
        // Fall back to keyword matching
        return m.keywordMatch(query, topK)
    }
    
    // Semantic search
    neighbors, scores, err := m.index.SearchKNN(queryEmb, topK)
    if err != nil {
        return nil, err
    }
    
    results := make([]MatchedSkill, len(neighbors))
    for i, idx := range neighbors {
        results[i] = MatchedSkill{
            Skill: m.skills[idx],
            Score: scores[i],
        }
    }
    
    return results, nil
}

func (m *SkillMatcher) keywordMatch(query string, topK int) ([]MatchedSkill, error) {
    queryLower := strings.ToLower(query)
    results := []MatchedSkill{}
    
    for _, skill := range m.skills {
        descLower := strings.ToLower(skill.Description)
        
        // Simple keyword scoring
        score := 0.0
        for _, word := range strings.Fields(queryLower) {
            if strings.Contains(descLower, word) {
                score += 1.0
            }
            if strings.Contains(skill.Name, word) {
                score += 2.0 // Name match is stronger
            }
        }
        
        if score > 0 {
            results = append(results, MatchedSkill{Skill: skill, Score: score})
        }
    }
    
    // Sort by score and return topK
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    if len(results) > topK {
        results = results[:topK]
    }
    
    return results, nil
}
```

### 11. Auto-Suggestion

Suggest relevant skills based on task analysis.

```go
// internal/skills/suggester.go

type SkillSuggester struct {
    matcher *SkillMatcher
    config  SuggestionConfig
}

type SuggestionConfig struct {
    Enabled          bool
    MinConfidence    float64
    MaxSuggestions   int
    AutoActivate     bool    // Activate high-confidence matches automatically
    AutoActivateMin  float64 // Minimum score for auto-activation
}

func (s *SkillSuggester) Suggest(query string) ([]SkillSuggestion, error) {
    matches, err := s.matcher.Match(query, s.config.MaxSuggestions)
    if err != nil {
        return nil, err
    }
    
    suggestions := []SkillSuggestion{}
    for _, m := range matches {
        if m.Score < s.config.MinConfidence {
            continue
        }
        
        suggestions = append(suggestions, SkillSuggestion{
            Skill:        m.Skill,
            Confidence:   m.Score,
            AutoActivate: s.config.AutoActivate && m.Score >= s.config.AutoActivateMin,
        })
    }
    
    return suggestions, nil
}
```

## Configuration

```yaml
skills:
  enabled: true
  
  # Skill discovery paths (in order of priority)
  paths:
    - "${SPIN_PROJECT}/.spin/skills"    # Project-specific skills
    - "${HOME}/.spin/skills"            # User skills
    - "/usr/share/spin/skills"          # System skills
    
  # Progressive disclosure
  progressive:
    metadata_budget: 2000      # Max tokens for all skill metadata
    instruction_budget: 5000   # Max tokens per activated skill
    
  # Matching and suggestions
  matching:
    enabled: true
    embeddings: true           # Use semantic matching
    min_confidence: 0.5
    max_suggestions: 3
    
  # Auto-activation
  auto_activate:
    enabled: false             # Require explicit activation
    min_confidence: 0.9        # If enabled, minimum score
    
  # Built-in skills
  builtin:
    enabled: true
    skills:
      - code-review
      - debugging
      - testing
      - refactoring
      - git-workflow
```

## Implementation Plan

### Phase 1: Core Infrastructure (Week 1-2)
1. Implement `SkillDiscovery` with YAML frontmatter parsing
2. Implement `SkillLoader` with caching
3. Add SKILL.md validation using skills-ref patterns
4. Create basic `SkillTool` (list, activate, reference)

### Phase 2: Integration (Week 3)
1. Implement `SkillInjector` for system prompt
2. Add skill metadata to agent startup
3. Create built-in skills (code-review, debugging, testing)
4. Add skill events for observability

### Phase 3: Matching (Week 4)
1. Implement `SkillMatcher` with keyword matching
2. Add optional semantic matching with embeddings
3. Build `SkillSuggester` for recommendations
4. Add skill usage tracking

### Phase 4: Polish (Week 5)
1. Create skill authoring documentation
2. Add skill validation CLI command
3. Implement skill caching and refresh
4. Performance optimization

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Context Size (specialized tasks) | 50K+ tokens | <10K tokens |
| Skill Activation Latency | N/A | <100ms |
| Skill Match Accuracy | N/A | 85%+ |
| Ecosystem Compatibility | 0 tools | Claude Code, Cursor compatible |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Skill not activated when needed | Medium | Suggestion system, good descriptions |
| Skill instructions too long | Medium | Progressive disclosure, reference files |
| Incompatible skill formats | Low | Validate against specification |
| User skill quality | Low | Validation, best practices docs |

## Examples

### Example 1: Code Review Task

**Query**: "Review the authentication code for security issues"

**Agent Flow**:
1. Skill suggester matches "code-review" (confidence: 0.92)
2. Agent activates: `skill(action='activate', name='code-review')`
3. Agent receives review instructions and criteria
4. Agent loads reference: `skill(action='reference', name='code-review', reference='references/CHECKLIST.md')`
5. Agent performs review following skill guidance

**Context Quarantine**: Review instructions loaded on-demand, not always in context.

### Example 2: Custom Workflow

**Query**: "Deploy the staging environment"

**User's Custom Skill** (`~/.spin/skills/deployment/SKILL.md`):
```yaml
---
name: deployment
description: Deploy applications to staging and production environments. Use for deploy, release, or environment management tasks.
---

# Deployment Skill

## Staging Deployment
1. Run tests: `make test`
2. Build image: `make docker-build`
3. Push to registry: `make docker-push`
4. Deploy: `kubectl apply -f k8s/staging/`
5. Verify: `kubectl get pods -n staging`
...
```

**Result**: Agent follows company-specific deployment process.

### Example 3: No Skill Needed

**Query**: "Read the README file"

**Agent Flow**:
1. Skill suggester finds no high-confidence match
2. Agent handles directly without skill activation
3. Minimal context overhead

## Comparison: Agent Skills vs Custom Sub-Agents

| Aspect | Agent Skills | Custom Sub-Agents |
|--------|--------------|-------------------|
| Complexity | Low (folder + markdown) | High (orchestration code) |
| Interoperability | Yes (open standard) | No (Spin-specific) |
| User Extensibility | Easy (create SKILL.md) | Hard (requires code) |
| Context Isolation | Progressive disclosure | Separate context windows |
| Maintenance | Community + Anthropic | Spin team only |
| Ecosystem | Growing (Cursor, VS Code, etc.) | None |

## References

- [Agent Skills Specification](https://agentskills.io/specification)
- [Agent Skills Integration Guide](https://agentskills.io/integrate-skills)
- [Example Skills Repository](https://github.com/anthropics/skills)
- [skills-ref Validation Library](https://github.com/agentskills/agentskills/tree/main/skills-ref)
- [LangChain Context Quarantine](https://github.com/langchain-ai/how_to_fix_your_context/blob/main/notebooks/3_context_quarantine.ipynb)
