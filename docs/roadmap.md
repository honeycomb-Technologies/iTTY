# Roadmap

## Current State

Last audited and corrected: 2026-03-28

The repository is now in a clean Phase 1 state:

- desktop daemon implemented in `daemon/`
- daemon docs and development docs aligned with code
- automated tests covering config, shell mutation, tmux parsing, API handlers, and startup failure behavior
- `ios/` now contains a verified source/resource/test scaffold copied from upstream Geistty
- there is still no buildable `ios/` app target yet
- `_upstream/geistty/` remains the reference for Xcode project wiring and upstream parity

## Phase 1

Phase 1 is the desktop daemon foundation. Its goal is a small, reliable daemon that can expose persistent tmux-backed terminal state to the future iOS client with a stable API and a clean configuration story.

### Phase 1 Status

Phase 1 is complete.

### Implemented

- Go module, build, race-test, lint, and cross-compile targets
- Daemon CLI with `start`, `status`, `auto on`, `auto off`, `sessions`, `version`, and `help`
- File-backed config load, validate, and atomic save at `~/.config/itty/config.toml`
- tmux client for installation detection, version lookup, session listing, detailed session inspection, and pane capture
- Strict tmux parsing with explicit malformed-output errors
- HTTP routes for `GET /health`, `GET /sessions`, `GET /sessions/{name}`, `GET /sessions/{name}/content`, `GET /config`, `PUT /config/auto`, and `GET /windows`
- Shell detection plus idempotent auto-wrap configure/unconfigure logic for bash, zsh, and fish
- Linux and macOS terminal window discovery primitives
- GitHub Actions workflow for daemon build, test, lint, and cross-compilation

### Phase 1 Verification

The Phase 1 correction sweep verified:

1. `make test` passes with race detection.
2. `make lint` passes.
3. `make cross` passes for Linux and macOS targets.
4. daemon startup fails immediately on a deterministic bind conflict.
5. daemon boots cleanly on an isolated temp config and serves the corrected HTTP routes.
6. `PUT /config/auto` updates both the persisted config and the managed shell rc block when run under an isolated temp home directory.

### Phase 1 Exit Notes

Phase 1 does not include:

- WebSocket streaming
- APNs device registration
- Tailscale control-plane integration
- a buildable iOS app target inside this repository

Those are Phase 2 and Phase 3 concerns, not open Phase 1 defects.

## Phase 2

Phase 2 is the actual iOS app integration:

- bring the Geistty fork into the real repo structure
- establish a buildable `ios/` app target
- add daemon discovery and session browser flows
- connect SSH + tmux attach flows to the daemon API
- document the iOS build and runtime architecture from the real code, not the intended shape

### Phase 2 Status

Phase 2 is in progress.

### Completed Prework

- audited and corrected the Geistty handoff in `docs/audit.md`
- verified a 49/49 upstream Swift source mapping and exported it to `ios/phase2-manifest.json`
- scaffolded `ios/iTTY/Sources`, `ios/iTTY/Assets.xcassets`, `ios/iTTY/Resources`, `ios/iTTYTests/UpstreamParity`, and `ios/iTTYUITests/UpstreamParity`
- added Linux-safe tooling to regenerate and verify the scaffold via `scripts/phase2_scaffold.py`
- added the first iTTY-owned Phase 2 source files for daemon browsing: machine models/store, daemon client, and machine/session browser views

### Remaining Phase 2 Work

- create and wire the real Xcode project/targets/schemes for iTTY
- rename app/module symbols where the scaffold still reflects upstream Geistty names
- implement daemon-backed session browser, discovery, and attach flows
- verify reconnect, tmux pane ownership, and Metal rendering behavior on simulator and device
- decide which upstream tests stay as parity tests and which become iTTY-specific tests

## Phase 3

Phase 3 is polish and resilience:

- WebSocket or equivalent real-time updates
- Tailscale device discovery and connectivity UX
- notifications and reconnect flows
- performance work, memory profiling, and operational hardening
- installer and platform-specific packaging quality
