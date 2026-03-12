// Package agentsmd provides AGENTS.md file discovery and loading.
//
// AGENTS.md files contain project-specific instructions for AI agents.
// This package discovers and loads these files, making their content
// available for injection into the agent's system prompt.
//
// Discovery order:
//  1. Working directory (./AGENTS.md)
//  2. Git repository root (if in a repo and different from workdir)
//  3. Parent directories up to filesystem root
//
// Usage:
//
//	cfg := &agentsmd.Config{
//	    Enabled: true,
//	    MaxSize: 100 * 1024, // 100KB
//	}
//	svc := agentsmd.NewService(cfg, workDir, gitRoot)
//	if err := svc.Load(ctx); err != nil {
//	    log.Printf("failed to load AGENTS.md: %v", err)
//	}
//	if svc.IsLoaded() {
//	    content := svc.Content()
//
// Use content in system prompt
//
//	}
package agentsmd
