package conversation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	osexec "os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/lsp"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ErrToolRegistryIsNil is a sentinel error.
var ErrToolRegistryIsNil = errors.New("tool registry is nil")

// ErrWebSearchNotConfigured indicates no search API provider is configured.
var ErrWebSearchNotConfigured = errors.New("web search requires a configured search API provider")

// ErrScreenshotNotConfigured indicates no headless browser is configured.
var ErrScreenshotNotConfigured = errors.New("screenshot capture requires a configured headless browser")

// registerIntegrationTools registers tools from MCP and Git integrations.
func (b *Builder) registerIntegrationTools(registry *tools.Registry) error {
	if registry == nil {
		return ErrToolRegistryIsNil
	}

	if b.mcpService != nil {
		err := b.registerMCPTools(registry)
		if err != nil {
			return fmt.Errorf("mcp tools: %w", err)
		}
	}

	if b.gitService != nil {
		err := b.registerGitTools(registry)
		if err != nil {
			return fmt.Errorf("git tools: %w", err)
		}
	}

	if b.lspManager != nil {
		b.registerLSPTools(registry)
	}

	b.registerWebTools(registry)

	return nil
}

// registerLSPTools registers LSP-backed code navigation tools.
func (b *Builder) registerLSPTools(registry *tools.Registry) {
	mgr := b.lspManager

	// DefinitionFinder delegates to Manager.ForFile().FindDefinition().
	findDef := func(ctx context.Context, filePath string, line, character int) ([]lsp.Location, error) {
		srv, srvErr := mgr.ForFile(ctx, filePath)
		if srvErr != nil {
			return nil, srvErr
		}

		return srv.FindDefinition(ctx, "file://"+filePath, line, character)
	}

	// ReferenceFinder delegates to Manager.ForFile().FindReferences().
	findRef := func(ctx context.Context, filePath string, line, character int) ([]lsp.Location, error) {
		srv, srvErr := mgr.ForFile(ctx, filePath)
		if srvErr != nil {
			return nil, srvErr
		}

		return srv.FindReferences(ctx, "file://"+filePath, line, character)
	}

	// SymbolRenamer delegates to Manager.ForFile().Rename().
	renameSym := func(ctx context.Context, filePath string, line, character int, newName string) (*lsp.WorkspaceEdit, error) {
		srv, srvErr := mgr.ForFile(ctx, filePath)
		if srvErr != nil {
			return nil, srvErr
		}

		return srv.Rename(ctx, "file://"+filePath, line, character, newName)
	}

	// SymbolSearcher delegates to Manager.ForFile().SearchSymbols().
	searchSym := func(ctx context.Context, filePath string, pattern string) ([]lsp.Symbol, error) {
		srv, srvErr := mgr.ForFile(ctx, filePath)
		if srvErr != nil {
			return nil, srvErr
		}

		return srv.SearchSymbols(ctx, "file://"+filePath, pattern)
	}

	_ = registry.Register(tools.NewFindSymbolToolWithSearch(findDef, searchSym))
	_ = registry.Register(tools.NewFindReferencesTool(findRef))
	_ = registry.Register(tools.NewRenameSymbolTool(renameSym))
}

// registerMCPTools registers all tools provided by MCP servers.
func (b *Builder) registerMCPTools(registry *tools.Registry) error {
	mcpTools := b.mcpService.GetTools()
	if len(mcpTools) == 0 {
		return nil
	}

	var names []string

	for _, t := range mcpTools {
		err := registry.Register(t)
		if err != nil {
			if b.logger != nil {
				b.logger.Warn("mcp tool register failed", "tool", t.Name(), "err", err)
			}

			continue
		}

		names = append(names, t.Name())
	}

	if len(names) > 0 && b.logger != nil {
		b.logger.Info("mcp tools registered", "tools", strings.Join(names, ", "))
	}

	return nil
}

// httpFetchTimeout is the timeout for HTTP fetch operations.
const httpFetchTimeout = 30 * time.Second

// maxFetchResponseBytes is the maximum response body size for URL fetching (5 MB).
const maxFetchResponseBytes = 5 * 1024 * 1024

// httpFetchClient is a shared HTTP client for web tool fetching (connection pooling).
var httpFetchClient = &http.Client{Timeout: httpFetchTimeout}

// registerWebTools registers web interaction tools (fetch, search, browser, screenshot).
func (b *Builder) registerWebTools(registry *tools.Registry) {
	// PageFetcher using shared HTTP client with connection pooling.
	fetcher := func(ctx context.Context, url string) (*tools.FetchResponse, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if reqErr != nil {
			return nil, fmt.Errorf("create request: %w", reqErr)
		}

		resp, doErr := httpFetchClient.Do(req)
		if doErr != nil {
			return nil, fmt.Errorf("fetch URL: %w", doErr)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxFetchResponseBytes))
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}

		return &tools.FetchResponse{
			StatusCode:  resp.StatusCode,
			ContentType: resp.Header.Get("Content-Type"),
			Body:        body,
		}, nil
	}

	// HTMLConverter using the built-in converter.
	converter := tools.ConvertHTML

	_ = registry.Register(tools.NewFetchURLTool(fetcher, converter))

	// WebSearcher — requires external search API.
	searcher := func(_ context.Context, _ string, _ int) ([]tools.SearchResult, error) {
		return nil, ErrWebSearchNotConfigured
	}

	_ = registry.Register(tools.NewWebSearchTool(searcher))

	// BrowserOpener using OS-specific open command.
	opener := func(ctx context.Context, url string) error {
		var cmd string

		switch runtime.GOOS {
		case "darwin":
			cmd = "open"
		case "windows":
			cmd = "start"
		default:
			cmd = "xdg-open"
		}

		return osexec.CommandContext(ctx, cmd, url).Start()
	}

	_ = registry.Register(tools.NewOpenBrowserTool(opener))

	// ScreenshotCapturer — requires headless browser.
	capturer := func(_ context.Context, _ string, _, _ int, _ bool) (string, error) {
		return "", ErrScreenshotNotConfigured
	}

	_ = registry.Register(tools.NewScreenshotTool(capturer))
}

// registerGitTools registers Git operation tools.
func (b *Builder) registerGitTools(registry *tools.Registry) error {
	return registry.Register(tools.NewGitOperationTool(b.gitService.GetIntegration()))
}

// buildToolRegistry constructs a complete tool registry with all standard and integration tools.
func (b *Builder) buildToolRegistry(exec *agent.Executor, securityService *safety.Service, env *agent.Environment) *tools.Registry {
	registry := b.toolRegistry
	if registry == nil {
		// Use shared factory to create base registry with configured tools.
		registry = tools.NewDefaultRegistry(env.WorkDir, env)

		if b.logger != nil {
			b.logger.Debug("created tool registry with builtins")
		}
	}

	var (
		validatorAdapt tools.CommandValidator
		shellCtxAdapt  tools.ShellContext
		execAdapt      tools.CommandExecutor
	)

	if securityService != nil {
		validatorAdapt = &validatorAdapter{securityService: securityService}
	}

	if b.shellService != nil {
		shellCtxAdapt = &shellContextAdapter{shellCtx: b.shellService.GetContext()}
	}

	if exec != nil {
		execAdapt = &executorAdapter{executor: exec}
	}

	// Replace shell_command tool with configured version (factory creates it with nil params)
	// Other tools (get_context, apply_patch, file_search, git_context) are already configured
	// by the factory, but we replace them again if they need different configuration.
	if validatorAdapt != nil || shellCtxAdapt != nil || execAdapt != nil {
		_ = registry.RegisterOrReplace(tools.NewShellCommandTool(validatorAdapt, shellCtxAdapt, execAdapt))
	}

	err := b.registerIntegrationTools(registry)
	if err != nil {
		if b.logger != nil {
			b.logger.Warn("integration tools registration failed", "err", err)
		}
	}

	return registry
}
