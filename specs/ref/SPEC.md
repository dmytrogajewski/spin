# Generalization spec — agent-harness march (quick-pass)

Scope: `internal/skills`, `internal/plugins`, `internal/contexteng/compact`, `internal/agent/frame`, `internal/protocol/a2a`, `internal/agent/child`, `internal/safety/hooks` (`dir.go`, `PluginScript`, `inherit.go`), `internal/agent/tool/runtime.go` (`applyUpdatedInput` only).

## Path containment

Two packages independently jail a relative path under a package root.

Shared core (extract once):

- `filepath.Abs(root)`
- `filepath.Join(absRoot, filepath.Clean(rel))`
- `filepath.Abs(joined)`
- `insideRoot(absRoot, absJoined)` — `filepath.Rel` must not be `".."` or a `".."` + separator prefix

Copies:

- `skills.insideRoot` — `internal/skills/resolve.go:73:1`
- `plugins.insideRoot` — `internal/plugins/contain.go:63:1`

Public wrappers that should keep distinct contracts:

- `skills.Resolve` — `internal/skills/resolve.go:14:1` — reject empty, absolute, and any `".."` component before join; no symlink walk
- `plugins.Contain` — `internal/plugins/contain.go:12:1` — require `"./"` prefix; if the path exists, `EvalSymlinks` and re-check `insideRoot`

Do not merge `Resolve` and `Contain`. Extract `InsideRoot` and optionally `JoinUnder(root, rel) (string, error)` for the four shared steps.

`pkg/alg/pathx.ResolvePath` is a different join (abs-or-workdir) and is not a replacement.

## Closed JSON schema keys

`plugin.json` and `mcp.json` both collect keys absent from a permitted set and sort them.

Copies:

- `collectUnknown` — `internal/plugins/manifest.go:49:1` — reports unknown fields as warnings
- `unknownMCPTopLevel` — `internal/plugins/mcpjson.go:75:1` — treats the first unknown as fatal

One function `unknownKeys(raw map[string]json.RawMessage, permitted map[string]struct{}) []string` replaces both loops. Callers keep their warn-vs-fail policy.

## First Part.Text

A2A echo and child crash/complete summaries both take the first non-empty `Part.Text`.

Copies:

- `firstText` — `internal/protocol/a2a/handler.go:192:1` — walks `Message.Parts`
- `firstArtifactText` — `internal/agent/child/spawn_send.go:69:1` — walks `Task.Artifacts`, then the same Part loop

Extract `FirstPartText(parts []Part) string` on `a2a`. `firstText` becomes `FirstPartText(message.Parts)`. `firstArtifactText` keeps the Artifact walk and calls `FirstPartText` per artifact.

Do not merge the Artifact walk into `firstText`. A Task is not a Message.

## Out of scope (scanned, no extract)

- Skill vs plugin `validateName` / `isNameByte` — overlapping charset walk, different Agent Skills vs Agent Plugins rules (leading hyphen vs alphanumeric ends, plugins allow `.`). Do not unify.
- Compact filters already share `decodeLines` / `encodeLines` / `renderGrouped` / `ignoreCmd` inside the package. No second copy outside compact.
- `hooks.DefaultGlobalDir`, `hooks.PluginScript` discovery, `hooks.CopyScripts`, and `applyUpdatedInput` are single-site adapters. Child already calls `CopyScripts` for inherit.
- `frame.FromMode` / `PhaseForMode` / `objectiveFor` / `formatFor` and `child.phaseForSpec` are domain lookups on different key spaces (mode vs spec name). Do not unify.
- `a2a.marshalResult` vs `mustMarshal` vs `frame.MarshalStable` — same `json.Marshal`, three error contracts (RPC error, `{}` fallback, wrapped error). Keep separate.
- `child.FindRepoBinary` walk-up and `fileExists` have no second copy in this scope.
