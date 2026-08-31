package main

import (
	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/ui/suggest"
)

// composeTUILine expands /skill and @file tokens before a turn or command.
func composeTUILine(line, workDir string) suggest.Result {
	return suggest.Expand(line, workDir, plugins.DiscoverCatalog(workDir, nil))
}
