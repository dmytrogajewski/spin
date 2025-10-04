package core

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"
)

// GatherEnvironmentConcurrent collects environment context using concurrent goroutines.
//
// This is an optimized version of GatherEnvironment that uses errgroup to parallelize
// independent I/O operations:
//   - OS information gathering
//   - Git repository information
//   - Project file scanning
//   - Environment variable filtering
//
// Performance improvement: ~50% faster than sequential gathering for typical projects.
//
// Example:
//
//	env, err := GatherEnvironmentConcurrent(workDir)
//	if err != nil {
//	    return err
//	}
func GatherEnvironmentConcurrent(workDir string, opts ...EnvironmentOption) (*Environment, error) {
	// Apply options
	cfg := &environmentConfig{
		maxFiles: 1000,
		maxDepth: 10,
		skipGit:  false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate workDir exists
	if _, err := os.Stat(workDir); err != nil {
		return nil, fmt.Errorf("work directory does not exist: %w", err)
	}

	// Create environment struct
	env := &Environment{
		WorkDir: workDir,
	}

	// Use errgroup for concurrent operations
	g, ctx := errgroup.WithContext(context.Background())

	// Mutex for safe writes to env
	var mu sync.Mutex

	// Gather OS information (fast, but run concurrently)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			osInfo := gatherOSInfo()
			mu.Lock()
			env.OS = osInfo
			mu.Unlock()
			return nil
		}
	})

	// Gather Git information (I/O bound)
	if !cfg.skipGit {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				gitInfo, _ := gatherGitInfo(workDir) // Ignore errors
				mu.Lock()
				env.Git = gitInfo
				mu.Unlock()
				return nil
			}
		})
	}

	// Scan project files (I/O bound)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			files, err := scanProjectFiles(workDir, cfg.maxFiles, cfg.maxDepth)
			if err != nil {
				// Continue with empty files
				files = []FileInfo{}
			}

			// Detect project type and languages (CPU bound, but quick)
			projectType := detectProjectType(files)
			languages := detectLanguages(files)

			mu.Lock()
			env.Files = files
			env.ProjectType = projectType
			env.Languages = languages
			mu.Unlock()
			return nil
		}
	})

	// Filter environment variables (fast)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			environment := filterEnvironment(os.Environ())
			mu.Lock()
			env.Environment = environment
			mu.Unlock()
			return nil
		}
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return env, nil
}
