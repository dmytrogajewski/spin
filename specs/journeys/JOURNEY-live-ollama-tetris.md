# JOURNEY-live-ollama-tetris: Live Ollama tetris generation

## Goal

Prove `spin exec` can drive a local Ollama model (`ornith-1.5:35b-262k`) through a multi-file web-game implementation and leave playable artifacts under `~/sources/testbed/spin/{test-run}`.

## Actor

A developer running live e2e against a local 35B coding model.

## Flow

1. Configure `~/.spin/spin.yaml` with provider `ollama`, model `ornith-1.5:35b-262k`, `context_window: 262144`.
2. Build the production `spin` binary (no `e2e_llm_test` tag).
3. For each test-run slug (`tetris-classic`, `tetris-hud`, `tetris-mobile`):
   - Create an empty workdir at `~/sources/testbed/spin/{slug}`.
   - Run `spin exec --auto-approve` with a tetris prompt.
   - Assert `index.html` exists and the web sources contain the required mechanics.
4. Open each game in a browser and confirm it renders and accepts input.

## Success

- All three live tests pass.
- Each run directory contains a playable `index.html` (plus JS/CSS as needed).
- Classic and HUD games show a falling piece and HUD; mobile includes on-screen controls.

## Implementation

- `tests/e2e/live/tetris_test.go`
- `tests/e2e/live/helpers_test.go`
- Run: `go test -tags e2e_ollama -v -count=1 -timeout 50m ./tests/e2e/live/`
