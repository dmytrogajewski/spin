package compact

// Default returns a pipeline with the spec per-command filter registry.
func Default() *Pipeline {
	pipeline := New()
	registerDefaults(pipeline)

	return pipeline
}

func registerDefaults(pipeline *Pipeline) {
	setAll(pipeline, treeCommands, filterTree)
	setAll(pipeline, readCommands, filterRead)
	setAll(pipeline, grepCommands, ignoreCmd(compactGrep))
	pipeline.SetFilter(cmdGitStatus, ignoreCmd(compactGitStatus))
	pipeline.SetFilter(cmdGitDiff, ignoreCmd(compactGitDiff))
	pipeline.SetFilter(cmdGitLog, ignoreCmd(compactGitLog))
	setAll(pipeline, gitConfirmCommands, ignoreCmd(compactGitConfirm))
	pipeline.SetFilter(cmdGoTest, ignoreCmd(compactGoTest))
	setAll(pipeline, failureCommands, ignoreCmd(compactFailures))
	setAll(pipeline, ruffCommands, ignoreCmd(compactRuff))
	pipeline.SetFilter(cmdDockerPS, ignoreCmd(compactDockerPS))
	setAll(pipeline, dedupCommands, ignoreCmd(compactDedup))
	pipeline.SetFilter(cmdJSON, filterJSON)
}

func setAll(pipeline *Pipeline, commands []string, filter Filter) {
	for _, command := range commands {
		pipeline.SetFilter(command, filter)
	}
}

var (
	treeCommands       = []string{"ls", "tree", "find"}
	readCommands       = []string{"cat", "read", "smart"}
	grepCommands       = []string{"grep", "rg"}
	gitConfirmCommands = []string{"git add", "git commit", "git push", "git pull"}
	failureCommands    = []string{"cargo test", "npm test", "pytest", "jest", "vitest", "playwright"}
	ruffCommands       = []string{"ruff", "ruff check"}
	dedupCommands      = []string{"log", "dedup"}
)

const (
	cmdGitStatus = "git status"
	cmdGitDiff   = "git diff"
	cmdGitLog    = "git log"
	cmdGoTest    = "go test"
	cmdDockerPS  = "docker ps"
	cmdJSON      = "json"
)
