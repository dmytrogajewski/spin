package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ToolPriority defines priority levels for tool selection.
type ToolPriority int

const (
	// PriorityCore is the highest priority for core builtin tools.
	PriorityCore ToolPriority = 100
	// PriorityStaticMCP is for statically configured MCP tools.
	PriorityStaticMCP ToolPriority = 50
	// PriorityDynamicMCP is for dynamically loaded MCP tools.
	PriorityDynamicMCP ToolPriority = 25
)

// ToolSelectionConfig configures dynamic tool selection behavior.
type ToolSelectionConfig struct {
	// Enabled controls whether dynamic tool selection is active.
	Enabled bool

	// MaxToolsPerTurn limits how many tools to include in each LLM turn.
	MaxToolsPerTurn int

	// MaxToolsPerSearch limits how many tools to return from each registry search.
	MaxToolsPerSearch int

	// MaxServersToLoad limits how many MCP servers to connect to per query.
	MaxServersToLoad int

	// SearchTimeout is the timeout for search operations.
	SearchTimeout time.Duration

	// CoreToolNames lists tools that are always included (highest priority).
	CoreToolNames []string
}

// DefaultToolSelectionConfig returns sensible defaults for tool selection.
func DefaultToolSelectionConfig() ToolSelectionConfig {
	return ToolSelectionConfig{
		Enabled:           true,
		MaxToolsPerTurn:   30,
		MaxToolsPerSearch: 10,
		MaxServersToLoad:  3,
		SearchTimeout:     10 * time.Second,
		CoreToolNames:     defaultCoreTools(),
	}
}

// defaultCoreTools returns the list of core tools that should always be included.
func defaultCoreTools() []string {
	return []string{
		// File operations.
		"Read", "Write", "Edit",
		// Search.
		"Glob", "Grep",
		// Execution.
		"Bash",
		// Agent tools.
		"Task", "TodoWrite",
		// Planning.
		"EnterPlanMode", "ExitPlanMode",
	}
}

// ScoredTool represents a tool with its relevance score and priority.
type ScoredTool struct {
	Tool       tools.Tool
	Score      float64
	Priority   ToolPriority
	Source     string // "core", "static_mcp", "dynamic_mcp".
	ServerPath string // For dynamic MCP tools.
	IsLoadable bool   // True if tool needs to be loaded.
}

// ToolSelectionResult contains the result of tool selection for a turn.
type ToolSelectionResult struct {
	// SelectedTools are the tools chosen for this turn.
	SelectedTools []tools.Tool
	// NewlyLoaded are tools that were loaded this turn (subset of SelectedTools).
	NewlyLoaded []tools.Tool
	// TotalSearched is the number of tools considered.
	TotalSearched int
	// Query is the search query used.
	Query string
	// OAuthRequired contains servers that failed to load due to OAuth requirements.
	OAuthRequired []OAuthRequiredServer
}

// OAuthRequiredServer represents a server that requires OAuth authentication.
type OAuthRequiredServer struct {
	ServerPath string
	ToolNames  []string
}

// ToolSelector handles dynamic tool discovery and selection.
// It implements the flow:
// 1. Search all connected registries
// 2. Join with static MCP tools
// 3. Filter based on trajectory context
// 4. Load dynamic tools as needed
// 5. Register loaded tools to runtime registry
// 6. Return prioritized tool list.
type ToolSelector struct {
	mcpService      *mcp.Service
	coreRegistry    *tools.Registry // Core builtin tools.
	runtimeRegistry *tools.Registry // Runtime registry where loaded tools are registered.
	emitter         *events.EventEmitter
	config          ToolSelectionConfig
	logger          *slog.Logger

	// Track loaded servers to avoid duplicates.
	loadedServers map[string]bool

	// Sticky tools: tools selected in previous turns stay available.
	stickyTools map[string]tools.Tool

	mu sync.RWMutex
}

// NewToolSelector creates a new ToolSelector.
func NewToolSelector(
	mcpService *mcp.Service,
	coreRegistry *tools.Registry,
	emitter *events.EventEmitter,
	config ToolSelectionConfig,
	logger *slog.Logger,
) *ToolSelector {
	return &ToolSelector{
		mcpService:    mcpService,
		coreRegistry:  coreRegistry,
		emitter:       emitter,
		config:        config,
		logger:        logger,
		loadedServers: make(map[string]bool),
		stickyTools:   make(map[string]tools.Tool),
	}
}

// SetRuntimeRegistry sets the runtime registry for dynamically loaded tools.
// Call this before SelectToolsForTurn to enable tool registration.
func (s *ToolSelector) SetRuntimeRegistry(registry *tools.Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runtimeRegistry = registry
}

// SelectToolsForTurn selects tools for a specific turn based on the query.
// This is the main entry point called from the agent loop.
//
// Flow:
// 1. Search all connected registries with Search method
// 2. Each registry returns top relevant tools (based on trajectory context)
// 3. All results joined WITH static MCP tools
// 4. Filter based on trajectory context and priorities
// 5. Load dynamic tools that were selected
// 6. Emit tool selection event.
func (s *ToolSelector) SelectToolsForTurn(ctx context.Context, query string, turn int) (*ToolSelectionResult, error) {
	if !s.config.Enabled {
		return s.fallbackSelection()
	}

	// Apply search timeout.
	searchCtx := ctx

	if s.config.SearchTimeout > 0 {
		var cancel context.CancelFunc

		searchCtx, cancel = context.WithTimeout(ctx, s.config.SearchTimeout)
		defer cancel()
	}

	// Step 1: Collect all candidate tools with scores.
	candidates := s.collectCandidates(searchCtx, query)

	if s.logger != nil {
		s.logger.DebugContext(ctx, "tool selection: candidates collected",
			"total", len(candidates),
			"query", query,
			"turn", turn)
	}

	// Step 2: Add sticky tools (tools from previous turns).
	candidates = s.addStickyTools(candidates)

	// Step 3: Sort by priority and score.
	s.sortCandidates(candidates)

	// Step 4: Select top N tools respecting limit.
	selected := s.selectTopTools(candidates)

	// Step 5: Load any dynamic tools that need loading.
	loadResult, err := s.loadDynamicTools(ctx, selected)
	if err != nil && s.logger != nil {
		s.logger.WarnContext(ctx, "some dynamic tools failed to load", "error", err)
	}

	// Step 6: Build final tool list (includes newly loaded tools).
	finalTools := s.buildFinalToolList(ctx, selected, loadResult.loaded)

	// Step 7: Update sticky tools.
	s.updateStickyTools(finalTools)

	// Step 8: Emit selection event.
	result := &ToolSelectionResult{
		SelectedTools: finalTools,
		NewlyLoaded:   loadResult.loaded,
		TotalSearched: len(candidates),
		Query:         query,
		OAuthRequired: loadResult.oauthRequired,
	}
	s.emitSelectionEvent(turn, result)

	// Step 9: Emit info event about newly loaded tools (user-visible).
	if len(loadResult.loaded) > 0 {
		s.emitLoadedToolsEvent(loadResult.loaded)
	}

	// Log OAuth failures for debugging only (not user-visible).
	if len(loadResult.oauthRequired) > 0 && s.logger != nil {
		serverNames := make([]string, len(loadResult.oauthRequired))
		for i, srv := range loadResult.oauthRequired {
			serverNames[i] = srv.ServerPath
		}

		s.logger.DebugContext(ctx, "some servers require OAuth authentication",
			"servers", serverNames,
			"hint", "configure these servers statically with OAuth credentials to use them")
	}

	if s.logger != nil {
		s.logger.InfoContext(ctx, "tool selection complete",
			"turn", turn,
			"selected", len(finalTools),
			"newly_loaded", len(loadResult.loaded),
			"oauth_required", len(loadResult.oauthRequired),
			"total_searched", len(candidates))
	}

	return result, nil
}

// collectCandidates gathers all candidate tools from all sources.
func (s *ToolSelector) collectCandidates(ctx context.Context, query string) []ScoredTool {
	var candidates []ScoredTool

	// 1. Add core tools (always highest priority).
	candidates = append(candidates, s.getCoreTools()...)

	// 2. Search all MCP registries.
	if s.mcpService != nil {
		mcpCandidates := s.searchMCPRegistries(ctx, query)
		candidates = append(candidates, mcpCandidates...)
	}

	return candidates
}

// getCoreTools returns core builtin tools with highest priority.
func (s *ToolSelector) getCoreTools() []ScoredTool {
	if s.coreRegistry == nil {
		return nil
	}

	coreSet := make(map[string]bool)
	for _, name := range s.config.CoreToolNames {
		coreSet[name] = true
	}

	var result []ScoredTool

	for _, t := range s.coreRegistry.List() {
		if coreSet[t.Name()] {
			result = append(result, ScoredTool{
				Tool:     t,
				Score:    1.0, // Max score for core tools.
				Priority: PriorityCore,
				Source:   "core",
			})
		}
	}

	return result
}

// searchMCPRegistries searches all MCP registries for relevant tools.
func (s *ToolSelector) searchMCPRegistries(ctx context.Context, query string) []ScoredTool {
	registryMgr := s.mcpService.GetRegistryManager()
	if registryMgr == nil {
		return nil
	}

	searchCtx := &mcp.SearchContext{
		DynamicLoadout: true,
	}

	var results []ScoredTool

	for _, reg := range registryMgr.All() {
		isDynamic := isRegistryDynamic(reg)
		foundTools := reg.Search(ctx, searchCtx, query, s.config.MaxToolsPerSearch)

		for i, t := range foundTools {
			scored := s.scoreRegistryTool(t, i, len(foundTools), isDynamic, reg)
			results = append(results, scored)
		}
	}

	return results
}

// isRegistryDynamic checks if a registry is a dynamic Smithery registry.
func isRegistryDynamic(reg mcp.Registry) bool {
	sr, ok := reg.(*mcp.SmitheryRegistry)

	return ok && sr.IsDynamic()
}

// scoreRegistryTool creates a ScoredTool from a registry search result.
func (s *ToolSelector) scoreRegistryTool(t tools.Tool, index, total int, isDynamic bool, reg mcp.Registry) ScoredTool {
	score := 1.0 - (float64(index) / float64(total+1))

	scored := ScoredTool{
		Tool:  t,
		Score: score,
	}

	if isDynamic {
		scored.Priority = PriorityDynamicMCP
		scored.Source = "dynamic_mcp"
		s.populateDynamicToolInfo(&scored, t)
	} else {
		scored.Priority = PriorityStaticMCP
		scored.Source = "static_mcp"
		s.checkIfAlreadyLoaded(&scored, reg)
	}

	return scored
}

// populateDynamicToolInfo fills in ServerPath and IsLoadable for dynamic tools.
func (s *ToolSelector) populateDynamicToolInfo(scored *ScoredTool, t tools.Tool) {
	if loadable, ok := t.(interface{ ServerPath() string }); ok {
		scored.ServerPath = loadable.ServerPath()
	}

	if loadable, ok := t.(interface{ IsLoadable() bool }); ok {
		scored.IsLoadable = loadable.IsLoadable()
	}
}

// checkIfAlreadyLoaded marks a tool as not loadable if its server is already loaded.
func (s *ToolSelector) checkIfAlreadyLoaded(scored *ScoredTool, reg mcp.Registry) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.loadedServers[reg.Name()]; exists {
		scored.IsLoadable = false
	}
}

// addStickyTools adds tools from previous turns that should persist.
func (s *ToolSelector) addStickyTools(candidates []ScoredTool) []ScoredTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build set of already-included tool names.
	existing := make(map[string]bool)
	for _, c := range candidates {
		existing[c.Tool.Name()] = true
	}

	// Add sticky tools that aren't already in candidates.
	for name, t := range s.stickyTools {
		if !existing[name] {
			candidates = append(candidates, ScoredTool{
				Tool:       t,
				Score:      0.5, // Medium score for sticky tools.
				Priority:   PriorityStaticMCP,
				Source:     "sticky",
				IsLoadable: false, // Already loaded.
			})
		}
	}

	return candidates
}

// sortCandidates sorts tools by priority (desc) then score (desc).
func (s *ToolSelector) sortCandidates(candidates []ScoredTool) {
	sort.Slice(candidates, func(i, j int) bool {
		// Higher priority first.
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		// Then higher score.
		return candidates[i].Score > candidates[j].Score
	})
}

// selectTopTools selects the top N tools respecting the limit.
func (s *ToolSelector) selectTopTools(candidates []ScoredTool) []ScoredTool {
	limit := s.config.MaxToolsPerTurn
	if limit <= 0 {
		limit = 30
	}

	if len(candidates) <= limit {
		return candidates
	}

	return candidates[:limit]
}

// loadDynamicToolsResult contains the result of loading dynamic tools.
type loadDynamicToolsResult struct {
	loaded        []tools.Tool
	oauthRequired []OAuthRequiredServer
}

// loadDynamicTools loads any selected dynamic tools that require server connection.
func (s *ToolSelector) loadDynamicTools(ctx context.Context, selected []ScoredTool) (*loadDynamicToolsResult, error) {
	result := &loadDynamicToolsResult{}

	// Group by server path.
	serverTools := s.groupByServerPath(selected)

	var loadErrors []error

	for serverPath, scoredTools := range serverTools {
		loaded, err := s.loadServer(ctx, serverPath)
		if err != nil {
			s.handleServerLoadError(ctx, err, serverPath, scoredTools, result)
			loadErrors = append(loadErrors, err)

			continue
		}

		result.loaded = append(result.loaded, loaded...)
	}

	if len(loadErrors) > 0 && s.logger != nil {
		s.logger.DebugContext(ctx, "some servers failed to load", "errors", len(loadErrors))
	}

	return result, nil
}

// groupByServerPath groups loadable tools by their server path.
func (s *ToolSelector) groupByServerPath(selected []ScoredTool) map[string][]ScoredTool {
	serverTools := make(map[string][]ScoredTool)

	for _, st := range selected {
		if st.IsLoadable && st.ServerPath != "" {
			serverTools[st.ServerPath] = append(serverTools[st.ServerPath], st)
		}
	}

	return serverTools
}

// handleServerLoadError logs the error and records OAuth failures.
func (s *ToolSelector) handleServerLoadError(ctx context.Context, err error, serverPath string, scoredTools []ScoredTool, result *loadDynamicToolsResult) {
	if isOAuthError(err) {
		toolNames := make([]string, len(scoredTools))
		for i, t := range scoredTools {
			toolNames[i] = t.Tool.Name()
		}

		result.oauthRequired = append(result.oauthRequired, OAuthRequiredServer{
			ServerPath: serverPath,
			ToolNames:  toolNames,
		})

		if s.logger != nil {
			s.logger.DebugContext(ctx, "server requires OAuth authentication",
				"server", serverPath, "tools", toolNames, "error", err.Error())
		}

		return
	}

	if s.logger != nil {
		s.logger.DebugContext(ctx, "failed to load server", "server", serverPath, "error", err)
	}
}

// isOAuthError checks if an error indicates OAuth/authentication is required.
func isOAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Check for common OAuth/auth error indicators.
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "Unauthorized") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "OAuth") ||
		strings.Contains(errStr, "oauth") ||
		strings.Contains(errStr, "authentication required") ||
		strings.Contains(errStr, "Authentication required")
}

// loadServer loads a specific MCP server and returns its tools.
func (s *ToolSelector) loadServer(ctx context.Context, serverPath string) ([]tools.Tool, error) {
	s.mu.Lock()
	if s.loadedServers[serverPath] {
		s.mu.Unlock()

		return nil, nil // Already loaded.
	}

	s.loadedServers[serverPath] = true
	s.mu.Unlock()

	if s.mcpService == nil {
		return nil, nil
	}

	registryMgr := s.mcpService.GetRegistryManager()
	if registryMgr == nil {
		return nil, nil
	}

	// Find a dynamic Smithery registry to use for loading.
	for _, reg := range registryMgr.All() {
		if sr, ok := reg.(*mcp.SmitheryRegistry); ok && sr.IsDynamic() {
			return sr.LoadServer(ctx, serverPath)
		}
	}

	return nil, nil
}

// buildFinalToolList extracts the actual tools from scored candidates,
// replacing loadable stubs with their actual loaded implementations.
func (s *ToolSelector) buildFinalToolList(ctx context.Context, selected []ScoredTool, newlyLoaded []tools.Tool) []tools.Tool {
	result := make([]tools.Tool, 0, len(selected)+len(newlyLoaded))
	seen := make(map[string]bool)

	if s.logger != nil {
		s.logger.DebugContext(ctx, "buildFinalToolList called",
			"selected_count", len(selected),
			"newly_loaded_count", len(newlyLoaded))
	}

	// First, add all non-loadable tools (core tools, static MCP tools).
	result = s.addNonLoadableTools(ctx, selected, seen, result)

	// Then add all newly loaded tools and register them.
	result = s.addAndRegisterNewTools(ctx, newlyLoaded, seen, result)

	if s.logger != nil {
		s.logger.DebugContext(ctx, "buildFinalToolList result", "total_tools", len(result))
	}

	return result
}

// addNonLoadableTools adds non-loadable tools from selected candidates, skipping stubs and duplicates.
func (s *ToolSelector) addNonLoadableTools(ctx context.Context, selected []ScoredTool, seen map[string]bool, result []tools.Tool) []tools.Tool {
	for _, st := range selected {
		if st.IsLoadable {
			if s.logger != nil {
				s.logger.DebugContext(ctx, "skipping loadable stub", "tool", st.Tool.Name(), "server", st.ServerPath)
			}

			continue
		}

		name := st.Tool.Name()
		if seen[name] {
			continue
		}

		seen[name] = true
		result = append(result, st.Tool)
	}

	return result
}

// addAndRegisterNewTools adds newly loaded tools and registers them to the runtime registry.
func (s *ToolSelector) addAndRegisterNewTools(ctx context.Context, newlyLoaded []tools.Tool, seen map[string]bool, result []tools.Tool) []tools.Tool {
	for _, t := range newlyLoaded {
		name := t.Name()
		if s.logger != nil {
			s.logger.DebugContext(ctx, "adding newly loaded tool", "tool", name)
		}

		if seen[name] {
			continue
		}

		seen[name] = true
		result = append(result, t)
		s.registerToRuntime(ctx, t)
	}

	return result
}

// registerToRuntime registers a tool to the runtime registry.
func (s *ToolSelector) registerToRuntime(ctx context.Context, t tools.Tool) {
	if s.runtimeRegistry == nil {
		return
	}

	err := s.runtimeRegistry.RegisterOrReplace(t)
	if err != nil {
		if s.logger != nil {
			s.logger.WarnContext(ctx, "failed to register dynamic tool", "tool", t.Name(), "error", err)
		}

		return
	}

	if s.logger != nil {
		s.logger.DebugContext(ctx, "registered dynamic tool to runtime", "tool", t.Name())
	}
}

// updateStickyTools updates the sticky tools set with currently selected tools.
func (s *ToolSelector) updateStickyTools(selectedTools []tools.Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range selectedTools {
		s.stickyTools[t.Name()] = t
	}
}

// emitSelectionEvent emits an event about tool selection for ACE and Terminal.
func (s *ToolSelector) emitSelectionEvent(turn int, result *ToolSelectionResult) {
	if s.emitter == nil {
		return
	}

	toolNames := make([]string, len(result.SelectedTools))
	for i, t := range result.SelectedTools {
		toolNames[i] = t.Name()
	}

	newlyLoadedNames := make([]string, len(result.NewlyLoaded))
	for i, t := range result.NewlyLoaded {
		newlyLoadedNames[i] = t.Name()
	}

	s.emitter.Emit(events.Event{
		Type: events.EventTypeToolSelection,
		Data: map[string]any{
			"turn":           turn,
			"query":          result.Query,
			"selected_tools": toolNames,
			"newly_loaded":   newlyLoadedNames,
			"total_searched": result.TotalSearched,
		},
	})
}

// emitLoadedToolsEvent emits an informational event about newly loaded dynamic tools.
// This informs the user about tools that were discovered and loaded for this turn.
func (s *ToolSelector) emitLoadedToolsEvent(loadedTools []tools.Tool) {
	if s.emitter == nil {
		return
	}

	// Group tools by server (extract server name from tool name pattern mcp_<server>_<tool>).
	serverTools := make(map[string][]string)

	for _, t := range loadedTools {
		name := t.Name()
		// Tool names are like mcp_brave_brave_web_search or mcp_bh-rat_context-awesome_get_awesome_items
		// Extract server part (everything between first mcp_ and the tool name).
		serverTools["dynamic"] = append(serverTools["dynamic"], name)
	}

	toolNames := make([]string, len(loadedTools))
	for i, t := range loadedTools {
		toolNames[i] = t.Name()
	}

	message := fmt.Sprintf("Loaded %d dynamic tools for this query.", len(loadedTools))
	details := "Tools: " + strings.Join(toolNames, ", ")

	s.emitter.Emit(events.Event{
		Type: events.EventInfo,
		Data: events.SystemEventData{
			Level:   "info",
			Message: message,
			Details: details,
		},
	})
}

// fallbackSelection returns all core tools when selection is disabled.
func (s *ToolSelector) fallbackSelection() (*ToolSelectionResult, error) {
	var allTools []tools.Tool

	if s.coreRegistry != nil {
		allTools = s.coreRegistry.List()
	}

	return &ToolSelectionResult{
		SelectedTools: allTools,
		NewlyLoaded:   nil,
		TotalSearched: len(allTools),
		Query:         "",
	}, nil
}

// GetLoadedServers returns the list of servers that have been loaded.
func (s *ToolSelector) GetLoadedServers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	servers := make([]string, 0, len(s.loadedServers))
	for server := range s.loadedServers {
		servers = append(servers, server)
	}

	return servers
}

// GetStickyTools returns the current set of sticky tools.
func (s *ToolSelector) GetStickyTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.stickyTools))
	for name := range s.stickyTools {
		names = append(names, name)
	}

	return names
}
