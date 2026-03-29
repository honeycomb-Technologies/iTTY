# Roadmap

## Current State

Last audited and corrected: 2026-03-29

The repository is now in a clean Phase 1 daemon state plus active Phase 2 iOS integration work:

- desktop daemon implemented in `daemon/`
- daemon docs and development docs aligned with code
- automated tests covering config, shell mutation, tmux parsing, API handlers, and startup failure behavior
- `ios/` contains a generated Xcode project with app, unit-test, and UI-test targets
- `ios/project.yml` is the source of truth for GhosttyKit wiring, package dependencies, and build settings
- `_upstream/geistty/` remains the reference for Xcode project parity, and `_upstream/ghostty/` plus `patches/ghostty/` drive GhosttyKit builds

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
- the full iOS client integration and GhosttyKit build pipeline

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

### Completed Mac Work

- full Geistty → iTTY rename across all 104 Swift files (0 Geistty references remain)
- created Xcode project via xcodegen with three targets (app, unit tests, UI tests)
- configured SPM dependencies (swift-nio-ssh fork with RSA, swift-nio-transport-services)
- created Info.plist with Face ID, background processing, Bonjour, custom fonts
- created entitlements with Keychain sharing and app groups
- verified `itty-N` session naming parity with `daemon/internal/shell/configure.go`
- verified `com.itty` bundle, logging, entitlement, and keychain identifiers across the iOS app
- added repo-owned Ghostty build patches in `patches/ghostty/`
- added `scripts/build_ghosttykit.sh` to apply patches, handle Metal Toolchain setup, build GhosttyKit, and install it into `ios/iTTY/Frameworks`
- re-enabled GhosttyKit wiring in `ios/project.yml`, including simulator-specific module maps and header search paths
- upgraded Zig to 0.15.2 for GhosttyKit builds and pinned the build script to prefer Homebrew Zig when available
- verified a clean Apple silicon simulator build with `xcodebuild -project ios/iTTY.xcodeproj -scheme iTTY -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build`

### Remaining Phase 2 Work

- verify Metal rendering and tmux pane ownership end-to-end on device (requires SSH credentials at first connect)
- decide which upstream tests stay as parity tests and which become iTTY-specific tests
- UI/UX polish sweep (visual design, onboarding, settings — Codex handoff target)

### Completed: Session Lifecycle

- daemon: added `POST /sessions` endpoint to create new detached tmux sessions
- daemon: added `NewSession(ctx, name)` method to tmux client
- iOS: added `DaemonClient.createSession(name:)` method
- iOS: added "+" button in SessionBrowserView to create sessions
- iOS: auto-creates and links ConnectionProfile when machines are discovered (Bonjour or Tailscale)
- iOS: discovered machines get default SSH profile (host, port 22, useTmux: true) ready for first-connect credential prompt

### Completed: Zero-Config Discovery

**Bonjour (local network — zero config):**
- daemon: added `bonjour` package that advertises `_itty._tcp` via `dns-sd` (macOS) or `avahi-publish` (Linux)
- daemon: Bonjour advertisement starts at daemon boot, stops on shutdown
- iOS: added `BonjourBrowser` service that browses for `_itty._tcp` on the local network
- iOS: `TailscaleDiscoveryService.refresh()` checks Bonjour discoveries, probes them, and auto-adds to MachineStore
- iOS: declared `_itty._tcp` in Info.plist NSBonjourServices
- result: open the app → daemon appears automatically with zero manual setup

**Tailscale peers (remote — bootstrapped from any known daemon):**
- daemon: added `Peer` struct and `Peers()` method to `tailscale.Client` (parses `tailscale status --json`)
- daemon: added `GET /peers` API endpoint with 503 (no tailscale), 502 (CLI error), and 200 (peer array) responses
- daemon: wired `tailscale.Client` into `api.NewServer` at startup
- daemon: auto-detects macOS Tailscale.app CLI at `/Applications/Tailscale.app/Contents/MacOS/Tailscale`
- iOS: added `TailscalePeer` model and `DaemonClient.peers()` method
- iOS: `refresh()` fetches peers from first reachable daemon, probes online non-self peers at `https://<dnsName>:443/health`, auto-adds discovered daemons
- iOS: added "Discovered on Tailnet" transient section in `TailscaleDiscoveryView`
- iOS: 8 unit tests covering the full autodiscovery flow
- zero API keys required

**Other:**
- daemon default port changed from 8080 to 3420
- verified device build and install on iPhone 15 Pro Max
- live-tested Bonjour auto-discovery from iPhone to MacBook daemon

## Phase 3

Phase 3 is polish and resilience:

- WebSocket or equivalent real-time updates
- notifications and reconnect flows
- performance work, memory profiling, and operational hardening
- installer and platform-specific packaging quality
