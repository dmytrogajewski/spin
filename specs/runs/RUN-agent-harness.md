# March Run: agent-harness

## Mode
march

## Starting condition
All 26 Agent Harness roadmap items are unchecked. Workspace pre-flight is green. No prior RUN log existed.

## Plan
1. Parse and validate Agent Skills
2. Discover the skill catalog and list it to the user
3. Activate a skill body with progressive disclosure
4. Parse Agent Plugins 1.0 with path containment
5. Load plugins — merge skills, isolate MCP failures
6. Load com.spin.agent extension hooks from plugins
7. Finish the hook runner contract
8. Wire every defined lifecycle hook event
9. Compact pipeline core
10. Compact command registry
11. Apply compact to shell exec and PreToolUse rewrite
12. Apply compact to built-in read, grep, glob, and ls
13. Compact status chip and operator escape
14. TaskFrame on every parent turn
15. Assemble retrieval on the turn path
16. A2A types and local JSON-RPC codec
17. Local A2A server process
18. Spawn process children from the parent
19. Subagent hooks and no silent drop on spawn
20. Agent task registry
21. Unified task view
22. Structured navigation index
23. TUI and ACP surfaces
24. Remote A2A HTTPS client and card allowlist
25. Parent shutdown cancels children
26. Operator documentation

## Decision defaults captured
| Decision point | Default |
|---|---|
| Test framework | Makefile / `go test` |
| New dependency | Prefer in-repo packages; reject new deps unless already in go.mod |
| Lint warning | Fix |
| Ambiguous DoD | Strictest reasonable reading |
| Journey filename | `JOURNEY-NNN-<heading-slug>.md` (march template) |

## Assumptions
- A1 `.agents/instructions/instr-journey.md` and `instr-implement.md` are absent; subagents use `.agents/skills/journey/SKILL.md` and `.agents/skills/implement/SKILL.md`.
- A2 Baseline test count is 3265 `Test*` symbols and 107 packages reporting `ok`.
- A3 GitNexus MCP is unavailable; subagents use grep/impact-by-search and record that.
- A4 Step 19 child hook copy is proven on the in-process Harness; OS `spin a2a` has no hook-script argv yet (no CLI contract in Step 17).
- A5 ACP `session/cancel` CancelAlls running A2A tasks but does not `Close` the conversation; `SESSION_END` stays on `Conversation.Close` only (Step 8 / 25).

## Timeline
- [seq:1] [discovery] roadmap=specs/agent-harness/ROADMAP.md, items=26 unchecked
- [seq:2] [pre-flight] make lint=ok, make test=ok (count=3265)
- [seq:3] [item 1] start
- [seq:4] [item 1] subagent done → JOURNEY-001-parse-and-validate-agent-skills.md, tests 3265→3269, make lint ok
- [seq:5] [item 1] verified ok
- [seq:6] [item 1] done
- [seq:7] [item 2] start
- [seq:8] [item 2] subagent done → JOURNEY-002-discover-skill-catalog.md, tests 3269→3283, make lint ok
- [seq:9] [item 2] verified ok
- [seq:10] [item 2] done
- [seq:11] [item 3] start
- [seq:12] [item 3] subagent done → JOURNEY-003-activate-skill-body.md, tests 3283→3307, make lint ok
- [seq:13] [item 3] verified ok
- [seq:14] [item 3] done
- [seq:15] [item 4] start
- [seq:16] [item 4] subagent done → JOURNEY-004-parse-agent-plugins.md, tests 3307→3329, make lint ok
- [seq:17] [item 4] verified ok
- [seq:18] [item 4] done
- [seq:19] [item 5] start
- [seq:20] [item 5] subagent done → JOURNEY-005-load-plugins-merge-skills.md, tests 3329→3345, make lint ok
- [seq:21] [item 5] verified ok (discover roots, plugin: source tags, mcp.json transports, MCP fail isolates skills, fatal plugin.json isolates plugin)
- [seq:22] [item 5] done
- [seq:23] [item 6] start
- [seq:24] [item 6] subagent done → JOURNEY-006-load-com-spin-agent-extension-hooks.md, tests 3345→3360, make lint ok
- [seq:25] [item 6] verified ok (DiscoverAgentHooks + PluginScripts cwd=plugin root; foreign dirs ignored)
- [seq:26] [item 6] done
- [seq:27] [item 7] start
- [seq:28] [item 7] subagent done → JOURNEY-007-finish-the-hook-runner-contract.md, tests 3360→3372, make lint ok
- [seq:29] [item 7] verified ok (DefaultGlobalDir via pathx; UpdatedInput full-object replace; no Join("~"))
- [seq:30] [item 7] done
- [seq:31] [item 8] start
- [seq:32] [item 8] subagent done → JOURNEY-008-wire-every-defined-lifecycle-hook-event.md, tests 3372→3381, make lint ok
- [seq:33] [item 8] verified ok (POST_TOOL_USE_FAILURE, PRE_COMPACT, STOP, SESSION_END; parent subset integration)
- [seq:34] [item 8] done
- [seq:35] [item 9] start
- [seq:36] [item 9] subagent done → JOURNEY-009-compact-pipeline-core.md, tests 3381→3394, make lint ok
- [seq:37] [item 9] verified ok (Apply identity/R12/R13/R14/R15; p99 fixture; no tokenizer)
- [seq:38] [item 9] done
- [seq:39] [item 10] start
- [seq:40] [item 10] subagent done → JOURNEY-010-compact-command-registry.md, tests 3394→3402, make lint ok
- [seq:41] [item 10] verified ok (Default registry + goldens for spec table; R11 named only)
- [seq:42] [item 10] done
- [seq:43] [generalize] quick-pass spawned (parallel; do not block)
- [seq:43b] [generalize] quick-pass at item 10 → insideRoot dup (skills/plugins); collectUnknown vs unknownMCPTopLevel; LIST.md + SPEC.md written under specs/ref/
- [seq:45] [item 11] subagent done → JOURNEY-011-apply-compact-to-shell-exec.md, tests 3402→3427, make lint ok
- [seq:46] [item 11] verified ok (shell Apply + R11 argv rewrite; SPIN_COMPACT=0; rtk fallback)
- [seq:47] [item 11] done
- [seq:48] [item 12] start
- [seq:49] [item 12] subagent done → JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md, tests 3427→3439, make lint ok
- [seq:50] [item 12] verified ok (read minimal + list R10 + grep R2/R3 + escape hatch)
- [seq:51] [item 12] done
- [seq:52] [item 13] start
- [seq:53] [item 13] subagent done → JOURNEY-013-compact-status-chip-and-operator-escape.md, tests 3439→3447, make lint ok
- [seq:54] [item 13] verified ok (status on/off + −N% ledger chip; /help escape)
- [seq:55] [item 13] done
- [seq:56] [item 14] start
- [seq:57] [item 14] subagent done → JOURNEY-014-taskframe-on-every-parent-turn.md, tests 3447→3462, make lint ok
- [seq:58] [item 14] verified ok (TaskFrame MarshalStable; dynamic Composer; mode→phase)
- [seq:59] [item 14] done
- [seq:60] [item 15] start
- [seq:61] [item 15] subagent done → JOURNEY-015-assemble-retrieval-on-the-turn-path.md, tests 3462→3469, make lint ok
- [seq:62] [item 15] verified ok (GetRetrievalPipeline + Assemble on turn 0; nil no-op)
- [seq:63] [item 15] done
- [seq:64] [item 16] start
- [seq:65] [item 16] subagent done → JOURNEY-016-a2a-types-and-local-json-rpc-codec.md, tests 3469→3498, make lint ok
- [seq:66] [item 16] verified ok (NDJSON A2A types + pipe round-trip; no HTTP)
- [seq:67] [item 16] done
- [seq:68] [item 17] start
- [seq:69] [item 17] subagent done → JOURNEY-017-local-a2a-server-process.md, tests 3498→3518, make lint ok
- [seq:70] [item 17] verified ok (spin a2a cobra; card+send; unix://; p99 < 200ms)
- [seq:71] [item 17] done
- [seq:72] [item 18] start
- [seq:73] [item 18] subagent done → JOURNEY-018-spawn-process-children-from-the-parent.md, tests 3518→3526, make lint ok
- [seq:74] [item 18] verified ok (process spawn + A2A send; pid; crash failed; events; Subagents config)
- [seq:75] [item 18] done
- [seq:76] [item 19] start
- [seq:77] [item 19] subagent done → JOURNEY-019-subagent-hooks-and-no-silent-drop.md, tests 3526→3538, make lint ok
- [seq:78] [item 19] verified ok (START veto no pid; STOP on exit; CopyScripts; A4: in-process child)
- [seq:79] [item 19] done
- [seq:80] [item 20] start
- [seq:81] [generalize] quick-pass spawned (item 20 cadence)
- [seq:81b] [generalize] quick-pass at item 20 → firstText vs firstArtifactText; LIST.md updated under specs/ref/
- [seq:82] [item 20] subagent done → JOURNEY-020-agent-task-registry.md, tests 3538→3564, make lint ok
- [seq:83] [item 20] verified ok (SpawnBackground + /tasks wait/cancel + persist + no-deadlock wait)
- [seq:84] [item 20] done
- [seq:85] [item 21] start
- [seq:86] [item 21] subagent done → JOURNEY-021-unified-task-view.md, tests 3564→3581, make lint ok
- [seq:87] [item 21] verified ok (kind=agent|shell typed ids; mixed list; shell cancel)
- [seq:88] [item 21] done
- [seq:89] [item 22] start
- [seq:90] [item 22] subagent done → JOURNEY-022-structured-navigation-index.md, tests 3581→3598, make lint ok
- [seq:91] [item 22] verified ok (nav records + navigate tool; R10 paths; live catalogs)
- [seq:92] [item 22] done
- [seq:93] [item 23] start
- [seq:94] [item 23] subagent done → JOURNEY-023-tui-and-acp-surfaces.md, tests 3598→3612, make lint ok
- [seq:95] [item 23] verified ok (SKILL/TASK/SUBAGENT/HOOK/COMPACT; tasks N/M; /agents; kind=a2a)
- [seq:96] [item 23] done
- [seq:97] [item 24] start
- [seq:98] [item 24] subagent done → JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md, tests 3612→3625, make lint ok
- [seq:99] [item 24] verified ok (a2a.allowlist; reject before dial; fake HTTPS send/get)
- [seq:100] [item 24] done
- [seq:101] [item 25] start
- [seq:102] [item 25] subagent done → JOURNEY-025-parent-shutdown-cancels-children.md, tests 3625→3647, make lint ok
- [seq:103] [item 25] verified ok (CancelAll on Close; stdin/socket EOF tests; ReapOnStart; ClearHome)
- [seq:104] [item 25] done
- [seq:105] [item 26] start
- [seq:106] [item 26] subagent done → JOURNEY-026-operator-documentation.md, tests 3647→3647, make lint ok
- [seq:107] [item 26] verified ok (five Diátaxis pages; README SEE ALSO; honesty notes)
- [seq:108] [item 26] done

## Completed
- Step 1 / JOURNEY-001-parse-and-validate-agent-skills — Parse/Validate SKILL.md (seq:[6])
- Step 2 / JOURNEY-002-discover-skill-catalog — Catalog + /skills (seq:[10])
- Step 3 / JOURNEY-003-activate-skill-body — skill tool + SKILL block (seq:[14])
- Step 4 / JOURNEY-004-parse-agent-plugins — plugin.json + contain (seq:[18])
- Step 5 / JOURNEY-005-load-plugins-merge-skills — plugin skills + isolated MCP (seq:[22])
- Step 6 / JOURNEY-006-load-com-spin-agent-extension-hooks — com.spin.agent hooks (seq:[26])
- Step 7 / JOURNEY-007-finish-the-hook-runner-contract — ~ expand + UpdatedInput (seq:[30])
- Step 8 / JOURNEY-008-wire-every-defined-lifecycle-hook-event — parent lifecycle emitters (seq:[34])
- Step 9 / JOURNEY-009-compact-pipeline-core — R12–R15 pipeline (seq:[38])
- Step 10 / JOURNEY-010-compact-command-registry — R1–R10 goldens (seq:[42])
- Step 11 / JOURNEY-011-apply-compact-to-shell-exec — shell compact + R11 (seq:[47])
- Step 12 / JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls — builtin compact (seq:[51])
- Step 13 / JOURNEY-013-compact-status-chip-and-operator-escape — status chip + /help (seq:[55])
- Step 14 / JOURNEY-014-taskframe-on-every-parent-turn — TaskFrame + mode phases (seq:[59])
- Step 15 / JOURNEY-015-assemble-retrieval-on-the-turn-path — Assemble on turn path (seq:[63])
- Step 16 / JOURNEY-016-a2a-types-and-local-json-rpc-codec — A2A NDJSON types (seq:[67])
- Step 17 / JOURNEY-017-local-a2a-server-process — spin a2a stdio/unix (seq:[71])
- Step 18 / JOURNEY-018-spawn-process-children-from-the-parent — OS process spawn (seq:[75])
- Step 19 / JOURNEY-019-subagent-hooks-and-no-silent-drop — SUBAGENT_* + inherit (seq:[79])
- Step 20 / JOURNEY-020-agent-task-registry — wait/list/cancel persist (seq:[84])
- Step 21 / JOURNEY-021-unified-task-view — kind=agent|shell (seq:[88])
- Step 22 / JOURNEY-022-structured-navigation-index — navigate index (seq:[92])
- Step 23 / JOURNEY-023-tui-and-acp-surfaces — TUI/ACP surfaces (seq:[96])
- Step 24 / JOURNEY-024-remote-a2a-https-client-and-card-allowlist — HTTPS allowlist (seq:[100])
- Step 25 / JOURNEY-025-parent-shutdown-cancels-children — quit cancel + reap (seq:[104])
- Step 26 / JOURNEY-026-operator-documentation — Diátaxis how-tos + references (seq:[108])

## Blocked (if hard-stop)

## Final Run Summary
- Mode: march
- Roadmap: specs/agent-harness/ROADMAP.md
- Items completed: 26/26
- Items skipped: 0
- Retries: 0
- Assumptions logged: 5 (A1–A5)
- Tests: 3265→3647 (+382)
- Status: complete
