package plugins

import (
	"encoding/json"
	"os"

	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

// AgentHook is a contained com.spin.agent hook script.
type AgentHook struct {
	// ScriptName is Event.ScriptName() (for example pre-tool-use).
	ScriptName string
	// Path is the contained absolute script path.
	Path string
	// Cwd is the plugin root used as the script working directory.
	Cwd string
}

// DiscoverAgentHooks finds contained hook scripts under com.spin.agent/.
func DiscoverAgentHooks(plugin Plugin) []AgentHook {
	found := make([]AgentHook, 0)

	for _, event := range hooks.AllEvents() {
		hook, ok := containedHook(plugin.Root, event.ScriptName())
		if !ok {
			hook, ok = extensionFileHook(plugin, event.ScriptName())
		}

		if !ok {
			continue
		}

		found = append(found, hook)
	}

	return found
}

// HookScripts converts discovered agent hooks into runner plugin scripts.
func HookScripts(loaded []Plugin) []hooks.PluginScript {
	scripts := make([]hooks.PluginScript, 0)

	for _, plugin := range loaded {
		for _, hook := range DiscoverAgentHooks(plugin) {
			scripts = append(scripts, hooks.PluginScript{
				Name: hook.ScriptName,
				Path: hook.Path,
				Cwd:  hook.Cwd,
			})
		}
	}

	return scripts
}

func containedHook(root, scriptName string) (AgentHook, bool) {
	rel := relPathPrefix + SpinAgentHooksDir + "/" + scriptName

	return containedRelHook(root, scriptName, rel)
}

func extensionFileHook(plugin Plugin, scriptName string) (AgentHook, bool) {
	raw, ok := plugin.Manifest.Extensions[SpinAgentExtension]
	if !ok {
		return AgentHook{}, false
	}

	var ext struct {
		Hooks json.RawMessage `json:"hooks"`
	}
	if json.Unmarshal(raw, &ext) != nil || len(ext.Hooks) == 0 {
		return AgentHook{}, false
	}

	return extensionHooksValue(plugin.Root, scriptName, ext.Hooks)
}

func extensionHooksValue(root, scriptName string, raw json.RawMessage) (AgentHook, bool) {
	var files map[string]string
	if json.Unmarshal(raw, &files) == nil {
		rel, hasRel := files[scriptName]
		if !hasRel {
			return AgentHook{}, false
		}

		return containedRelHook(root, scriptName, rel)
	}

	var dir string
	if json.Unmarshal(raw, &dir) != nil || dir == "" {
		return AgentHook{}, false
	}

	return containedRelHook(root, scriptName, dir+"/"+scriptName)
}

func containedRelHook(root, scriptName, rel string) (AgentHook, bool) {
	path, err := Contain(root, rel)
	if err != nil {
		return AgentHook{}, false
	}

	info, statErr := os.Stat(path)
	if statErr != nil || !info.Mode().IsRegular() {
		return AgentHook{}, false
	}

	return AgentHook{ScriptName: scriptName, Path: path, Cwd: root}, true
}
