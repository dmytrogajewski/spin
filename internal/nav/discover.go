package nav

import "github.com/dmytrogajewski/spin/internal/skills"

// DiscoverOpts selects live skill roots. Plugin rows are injected by the caller
// (conversation) so this package does not import plugins (tools import cycle).
type DiscoverOpts struct {
	WorkDir      string
	HomeDir      string
	PluginSkills []skills.PluginSkill
	Plugins      []PluginRow
	Sessions     SessionSource
	Symbols      SymbolSource
	Peers        []Peer
}

// Discover builds an index from the live skill catalog plus injected rows.
func Discover(opts DiscoverOpts) *Index {
	return New(Sources{
		Skills: skills.Discover(skills.Options{
			WorkDir:      opts.WorkDir,
			HomeDir:      opts.HomeDir,
			PluginSkills: opts.PluginSkills,
		}),
		Plugins:  opts.Plugins,
		Sessions: opts.Sessions,
		Peers:    opts.Peers,
		Symbols:  opts.Symbols,
		WorkDir:  opts.WorkDir,
	})
}

// Live discovers skills using the process home directory.
func Live(workDir string, sessions SessionSource) *Index {
	return Discover(DiscoverOpts{
		WorkDir:  workDir,
		HomeDir:  skills.OptionsFor(workDir).HomeDir,
		Sessions: sessions,
	})
}
