## Dedup opportunities

1. internal/skills

   Function: insideRoot
   Position: internal/skills/resolve.go:73:1
   Findings: (a)(d) Byte-identical path-jail predicate: filepath.Rel(root, candidate) must not be ".." or a ".." prefix. A second copy lives in plugins. Extract one helper (pathx.InsideRoot or keep one unexported and share via a small path package) and delete the other.
   Could replace:
     - internal/plugins/contain.go:63:1:insideRoot

   Function: Resolve
   Position: internal/skills/resolve.go:14:1
   Findings: (a)(c) Join-under-root skeleton is the same as plugins.Contain: Abs(root), Join+Clean(rel), Abs(joined), insideRoot. Resolve additionally rejects empty/absolute/".." components before join. Contain additionally requires a "./" prefix and EvalSymlinks when the path exists. A shared JoinUnder(root, rel) (abs, error) would carry the common four steps; the two public functions keep their distinct admission rules.
   Could replace:
     - (partial) internal/plugins/contain.go:12:1:Contain — shared join+jail only, not the "./" or symlink policy

2. internal/plugins

   Function: insideRoot
   Position: internal/plugins/contain.go:63:1
   Findings: (a)(d) Same predicate as skills.insideRoot. See skills entry.
   Could replace:
     - internal/skills/resolve.go:73:1:insideRoot

   Function: Contain
   Position: internal/plugins/contain.go:12:1
   Findings: (a)(c) See skills.Resolve. Contain is the stricter public API (plugin-relative "./" + symlink stays inside root). Do not replace Resolve with Contain or the reverse; only extract the shared join+jail.
   Could replace:
     - (partial) internal/skills/resolve.go:14:1:Resolve — shared join+jail only

   Function: collectUnknown
   Position: internal/plugins/manifest.go:49:1
   Findings: (b)(c) Closed-schema unknown-key walk: iterate map[string]json.RawMessage, skip keys in a permitted set, sort leftovers. unknownMCPTopLevel is the same loop with a different permitted set. One helper unknownKeys(raw, permitted) []string with no type parameters needed beyond the existing map types.
   Could replace:
     - internal/plugins/mcpjson.go:75:1:unknownMCPTopLevel

   Function: unknownMCPTopLevel
   Position: internal/plugins/mcpjson.go:75:1
   Findings: (d) Replaceable by collectUnknown if collectUnknown takes the permitted set as an argument (mcpjson already has permittedMCPFields).
   Could replace:
     - internal/plugins/manifest.go:49:1:collectUnknown

3. internal/protocol/a2a

   Function: firstText
   Position: internal/protocol/a2a/handler.go:192:1
   Findings: (a)(c) First non-empty Part.Text over Message.Parts. child.firstArtifactText repeats the same inner loop over each Artifact.Parts. Extract FirstPartText(parts []Part) string (no type parameters). firstText becomes FirstPartText(message.Parts).
   Could replace:
     - (partial) internal/agent/child/spawn_send.go:69:1:firstArtifactText — shared Part.Text scan only; Artifact walk stays in child

4. internal/agent/child

   Function: firstArtifactText
   Position: internal/agent/child/spawn_send.go:69:1
   Findings: (d) Outer walk is Task.Artifacts; inner loop is firstText. Keep the Artifact walk; replace the Part scan with FirstPartText.
   Could replace:
     - internal/protocol/a2a/handler.go:192:1:firstText
