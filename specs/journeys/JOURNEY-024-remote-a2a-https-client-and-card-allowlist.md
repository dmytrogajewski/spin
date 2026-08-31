# JOURNEY-024-remote-a2a-https-client-and-card-allowlist: Remote A2A HTTPS client and card allowlist

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Remote A2A HTTPS client and card allowlist

## 1. Journey

When **Alex already runs local `spin a2a` children over NDJSON and wants to talk to a published remote peer** I want **the same A2A client (Card, Task, Message) over an HTTPS JSON-RPC binding, fetching an Agent Card only when the card URL is on `a2a.allowlist`** so I **can send `message/send` and poll `tasks/get` without a second type stack, without changing ACP, and without opening an SSRF hole**.

## 2. CJM

Alex already has one `internal/protocol/a2a` type package and a local NDJSON client/server (Steps 16–18). Local children still spawn as OS processes. ACP remains the IDE host protocol. There is no HTTPS binding and no card-URL allowlist. This journey adds a remote binding only: GET the Agent Card, POST JSON-RPC with slash methods, reject any URL that is not on an explicit config allowlist before a dial, and refuse redirects that leave the list. It does **not** change parent shutdown (Step 25) or write operator how-tos (Step 26). Default allowlist is empty (no remote cards). Tests use `httptest` TLS, not the public internet.

Assumption: slash method names from the SPEC (`message/send`, `tasks/get`) stay the wire names; HTTPS is transport only. Assumption: allowlist entries are exact `https://` card URLs; empty allowlist forbids every remote fetch. Assumption: the JSON-RPC endpoint is the card URL (or a `supportedInterfaces` URL that is also on the list). Assumption: ACP files stay untouched.

### Phase 1: Name the allowlist

**User Intent:** Put remote card URLs in config so spin will not dial anything else.

**Actions:** Add `a2a.allowlist` to V2. Leave it empty by default. List one HTTPS card URL. Load config.

**Pain / Risk:** A missing key is treated as “allow all”. HTTP or relative URLs sneak onto the list. Allowlist lives only in code and never in YAML. Validation of the rest of V2 breaks when the new section is absent.

**Success Signal:** Default config has an empty allowlist. YAML `a2a.allowlist` unmarshals. Non-HTTPS entries fail validation. Existing V2 without an `a2a` key still validates.

### Phase 2: Fetch a card only when listed

**User Intent:** Resolve a remote Agent Card without opening an arbitrary URL.

**Actions:** Call the HTTPS client with an allowlisted card URL against a fake TLS server. Repeat with an off-list URL. Follow a redirect to an off-list host.

**Pain / Risk:** Off-list URL is dialed then rejected (SSRF already happened). Client follows redirects to metadata or intranet hosts. Tests hit the public internet. Card JSON uses a second type stack.

**Success Signal:** Off-list URL returns an allowlist error and a transport probe proves no RoundTrip. Redirect off-list is rejected. On-list GET returns the same `AgentCard` type as local children.

### Phase 3: Speak JSON-RPC on HTTPS

**User Intent:** Drive the remote peer with the same methods as a local child.

**Actions:** `message/send` a user text part. `tasks/get` the returned id. Confirm local stdio spawn still works.

**Pain / Risk:** HTTPS client invents PascalCase methods. Local NDJSON client is rewritten. ACP notification shapes change. Child stdio spawn regresses. Fake server is HTTP-only and the “HTTPS” claim is untested.

**Success Signal:** `message/send` and `tasks/get` succeed against `httptest.NewTLSServer`. Local `child.StartSpec` tests still pass. No ACP package edits.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| No remote binding; only NDJSON pipes | 3 | HTTPS JSON-RPC on the same types |
| Any URL would be a dial (SSRF) | 1–2 | Explicit `a2a.allowlist`; reject before dial |
| Dual Card/Task packages | 2–3 | One `internal/protocol/a2a` types package |
| Redirect to an off-list host | 2 | `CheckRedirect` refuses off-list Location |
| Local children look like they need HTTP | 3 | Stdio spawn stays the local path |

### North Star Summary

Alex lists a card URL under `a2a.allowlist`. Spin fetches that card over HTTPS, posts `message/send` and `tasks/get` with the same types used for local children, and never dials a URL that is not on the list (including redirect targets). An empty allowlist means no remote cards. ACP is unchanged. Local stdio children still spawn and answer.

### Stressors

1. Empty `a2a.allowlist` — every remote card URL is rejected before dial (safe default).
2. Off-allowlist URL — error is allowlist/not-listed; a `RoundTrip` probe is never called.
3. `http://` card URL — rejected (HTTPS only), even if the string is listed.
4. Redirect to a host/path not on the allowlist — request is not followed (SSRF).
5. Redirect to another allowlisted HTTPS URL — follow is permitted.
6. Card JSON is the existing `AgentCard` type (`supportedInterfaces`, camelCase); no second card struct.
7. `message/send` + `tasks/get` against `httptest.NewTLSServer` — both succeed; no public internet.
8. Local stdio `child.StartSpec` / `Process.Send` still complete (Step 18 does not regress).
9. ACP package is not imported and not edited (host protocol stays ACP).
10. JSON-RPC methods stay slash form (`message/send`, `tasks/get`), not PascalCase.
11. Card `supportedInterfaces` RPC URL that is not on the allowlist — rejected before that dial.
12. Invalid allowlist entry (no scheme/host) fails config validation.
13. Context cancel during POST — call returns `ctx.Err()` and does not hang.
14. Malformed JSON-RPC error from the peer — surfaced as `RPCError`, not a panic.
15. Self-signed `httptest` TLS is trusted only via the test client; production client still uses system TLS.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Operator lists one HTTPS card URL and a test client can fetch it
- [x] Empty config stays safe (no remotes) without extra flags

### Onboarding Clarity
- [x] Config key is `a2a.allowlist` (or equivalent documented on the type)
- [x] Off-list error names the URL and the allowlist rule

### Production-Ready Defaults
- [x] Default allowlist is empty (deny remotes)
- [x] Feature works for local children with zero A2A config

### Golden Path Quality
- [x] Allowlisted card GET + `message/send` + `tasks/get` succeed on a fake HTTPS server
- [x] Returned Task id from send matches `tasks/get`

### Decision Load
- [x] Only decision is which card URLs to list
- [x] No extra auth scheme required for this step (allowlist only)

### Progressive Complexity
- [x] Local stdio path unchanged when allowlist is empty
- [x] HTTPS client is opt-in via allowlist entries

### Error Quality
- [x] Off-list error is distinct from transport failure
- [x] Non-HTTPS URL is rejected with a scheme error

### Failure Safety
- [x] Failed remote dial does not kill local children
- [x] Redirect off-list does not leak the request body to the next host

### Runtime Transparency
- [x] Client exposes the fetched `AgentCard` (same `Card()` idea as local)
- [x] No silent fallback from HTTPS to NDJSON

### Debuggability
- [x] Tests record allowlist vs requested URL
- [x] Fake server is in-process (`httptest`), inspectable

### Cross-Surface Consistency
- [x] Same method names as local NDJSON client
- [x] Same Task/Message JSON tags

### Workflow Consistency
- [x] Config section follows V2 `mapstructure`/`yaml` tags
- [x] Child package can dial remote using the same allowlist slice

### Change Safety
- [x] Missing `a2a` key does not change other V2 validation
- [x] ACP sources are not rewritten

### Experimentation Safety
- [x] Tests never call the public internet
- [x] Allowlist can be emptied to disable remotes

### Interaction Latency
- [x] One GET (card) then POSTs for RPC; no extra discovery hop
- [x] Context is honored on each HTTP round trip

### Developer Feedback Speed
- [x] Unit tests fail on off-list dial and on RPC mismatch
- [x] Local stdio tests remain the regression net for children

### Team Scale
- [x] Allowlist lives in YAML and can be committed
- [x] Exact URL strings make reviews obvious

### System Scale
- [x] Binding is a new file; types stay in one package
- [x] Additional remotes are more allowlist rows, not new types

### Right Behavior by Default
- [x] Deny by default (empty allowlist)
- [x] HTTPS required; HTTP is not a fallback

### Anti-Bypass Design
- [x] Allowlist check runs before `http.Client` transport
- [x] Redirects re-check the Location against the same list

## 4. Tests

### TC-01: default_allowlist_empty

**Given** `DefaultV2()`.
**When** the A2A section is read.
**Then** `a2a.allowlist` is empty and `Validate` still passes.

### TC-02: yaml_unmarshals_allowlist

**Given** YAML with `a2a.allowlist` containing one `https://` URL.
**When** the document is unmarshaled into `V2`.
**Then** that URL is present on `cfg.A2A.Allowlist`.

### TC-03: allowlist_rejects_http

**Given** an allowlist entry with scheme `http`.
**When** `Validate` runs.
**Then** validation fails and the error names `a2a.allowlist`.

### TC-04: off_list_rejected_before_dial

**Given** a card URL that is not on the allowlist and an HTTP transport that fails the test if used.
**When** the HTTPS client is asked to fetch that card.
**Then** the error is not-allowlisted and the transport is unused.

### TC-05: on_list_fetches_card

**Given** a TLS `httptest` server that serves an `AgentCard` and that server URL on the allowlist.
**When** the client fetches the card.
**Then** `Card().Name` matches the fixture and types are the existing `AgentCard`.

### TC-06: message_send_and_tasks_get

**Given** the same fake HTTPS A2A server.
**When** the client calls `message/send` then `tasks/get` with the returned id.
**Then** both succeed and the task id matches.

### TC-07: redirect_off_list

**Given** an allowlisted URL that 302s to a different host not on the list.
**When** the client fetches the card.
**Then** the call fails with not-allowlisted and the off-list host is not contacted.

### TC-08: local_stdio_children_still_work

**Given** existing `internal/agent/child` stdio spawn tests.
**When** the suite runs.
**Then** `StartSpec` / `Send` still pass (Step 18).

### TC-09: empty_allowlist_blocks_remote

**Given** a valid HTTPS card URL and an empty allowlist.
**When** the client is constructed.
**Then** the URL is rejected before dial.

### TC-10: child_dial_remote_uses_allowlist

**Given** `child.DialRemote` and a fake HTTPS server.
**When** the URL is listed, send/get succeed; when it is not, the call fails before dial.
**Then** the child package is a consumer of the same binding.

### TC-11: rpc_url_must_be_listed

**Given** a card whose `supportedInterfaces` URL is a different origin not on the allowlist.
**When** the client would POST JSON-RPC there.
**Then** the POST is not sent.

### TC-12: acp_untouched

**Given** the Step 24 diff.
**When** files under `internal/protocol/acp` are inspected.
**Then** they are unchanged.

## Acceptance Criteria

- Config `a2a.allowlist` (or equivalent) is required for remote cards
- Off-allowlist URL is rejected before dial
- `message/send` + `tasks/get` succeed against a fake HTTPS A2A server in tests
- Local stdio children still work
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [Step 24](../agent-harness/ROADMAP.md)
- Implementation files: `internal/protocol/a2a/http.go`, `internal/config/config_v2.go`, `internal/agent/child/remote.go`
- Test files: `internal/protocol/a2a/http_test.go`, `internal/config/config_v2_test.go`, `internal/agent/child/remote_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md` — this journey
- `internal/protocol/a2a/http.go` — HTTPS JSON-RPC binding, allowlist, card GET, `message/send` + `tasks/get`
- `internal/protocol/a2a/http_test.go` — `httptest.NewTLSServer` card, RPC, redirect, and before-dial probes
- `internal/agent/child/remote.go` — `DialRemote` over the same binding
- `internal/agent/child/remote_test.go` — off-list reject and send/get

Files modified:
- `internal/config/config_v2.go` — `A2AV2.Allowlist`, `Allows`, HTTPS validation, `DefaultV2` empty list
- `internal/config/config_v2_test.go` — default empty, YAML unmarshal, HTTP reject, `Allows`
- `internal/config/loader_v2.go` — `a2a.allowlist` env bind
- `internal/protocol/a2a/card.go` — package comment names the HTTPS binding
- `specs/agent-harness/ROADMAP.md` — Step 24 DoD and traceability
