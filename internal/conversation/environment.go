package conversation

import (
	"fmt"

	"github.com/dmytrogajewski/spin/internal/agent"
)

// gatherEnvironmentContext gathers environment information for the given working directory.
func (b *Builder) gatherEnvironmentContext(workDir string) (*agent.Environment, error) {
	opts := b.buildEnvironmentOptions()
	env, err := agent.GatherEnvironment(workDir, opts...)
	if err != nil {
		return nil, fmt.Errorf("environment(%s): %w", workDir, err)
	}
	b.enrichEnvironmentWithIntegrations(env)
	return env, nil
}

// buildEnvironmentOptions constructs environment options from configuration.
func (b *Builder) buildEnvironmentOptions() []agent.EnvironmentOption {
	if b.cfg == nil {
		return nil
	}
	var opts []agent.EnvironmentOption
	if b.cfg.MaxFiles > 0 {
		opts = append(opts, agent.WithMaxFiles(b.cfg.MaxFiles))
	}
	if b.cfg.MaxDepth > 0 {
		opts = append(opts, agent.WithMaxDepth(b.cfg.MaxDepth))
	}
	if b.cfg.SkipGit {
		opts = append(opts, agent.WithSkipGit(true))
	}
	return opts
}

// enrichEnvironmentWithIntegrations adds context from Git and Shell integrations.
func (b *Builder) enrichEnvironmentWithIntegrations(env *agent.Environment) {
	if b.gitService != nil && b.gitService.IsRepository() {
		b.addGitContext(env)
	}
	if b.shellService != nil && b.shellService.IsEnabled() {
		b.addShellContext(env)
	}
}

// addGitContext enriches environment with Git repository information.
func (b *Builder) addGitContext(env *agent.Environment) {
	info := b.gitService.GetContextInfo()
	set := func(k, v string) { env.Environment[k] = v }

	set("git_enabled", boolString(info.GitEnabled))
	set("is_repo", boolString(info.IsRepo))
	if !info.IsRepo {
		if b.logger != nil {
			b.logger.Debug("git context: not a repository")
		}
		return
	}

	if info.Branch != "" {
		set("branch", info.Branch)
	}
	if info.Remote != "" {
		set("remote", info.Remote)
	}
	if info.Commit != "" {
		set("commit", info.Commit)
	}
	set("is_clean", boolString(info.IsClean))

	if info.ModifiedFiles > 0 {
		set("modified_files", fmt.Sprintf("%d", info.ModifiedFiles))
	}
	if info.UntrackedFiles > 0 {
		set("untracked_files", fmt.Sprintf("%d", info.UntrackedFiles))
	}
	if info.Ahead > 0 {
		set("ahead", fmt.Sprintf("%d", info.Ahead))
	}
	if info.Behind > 0 {
		set("behind", fmt.Sprintf("%d", info.Behind))
	}
	if info.Detached {
		set("detached", "true")
	}

	if b.logger != nil {
		b.logger.Debug("git context added", "branch", info.Branch, "clean", info.IsClean)
	}
}

// addShellContext enriches environment with Shell context information.
func (b *Builder) addShellContext(env *agent.Environment) {
	info := b.shellService.GetContextInfo()
	set := func(k, v string) { env.Environment[k] = v }

	set("shell_enabled", boolString(info.ShellEnabled))
	if !info.ShellEnabled {
		return
	}
	if info.Shell != "" {
		set("shell", info.Shell)
	}
	if info.ShellPath != "" {
		set("shell_path", info.ShellPath)
	}
	if b.logger != nil {
		b.logger.Debug("shell context added", "shell", info.Shell)
	}
}

// boolString converts a boolean to "true" or "false" string.
func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
