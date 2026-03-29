# iTTY — Development Guide

## Project Overview

iTTY targets a two-component persistent remote terminal system:
- **Desktop Daemon** (`/daemon/`): Go binary that manages tmux sessions and exposes them via REST API
- **iOS App** (`/ios/`): Swift/SwiftUI app powered by libghostty (Ghostty terminal engine)

## Repository Structure

```
iTTY/
├── daemon/       # Desktop daemon — Go
├── ios/          # iOS app — Swift/SwiftUI + GhosttyKit
│   ├── project.yml           # xcodegen spec (generates iTTY.xcodeproj)
│   ├── iTTY.xcodeproj/       # Generated — do not edit directly
│   ├── iTTY/Sources/         # App source (57 Swift files)
│   ├── iTTYTests/            # Unit tests
│   └── iTTYUITests/          # UI tests
├── docs/         # Documentation, roadmap, and engineering standards
└── _upstream/    # Upstream references (gitignored, not part of build)
```

Current reality:
- `ios/` has a real Xcode project with three targets (app, unit tests, UI tests)
- SPM dependencies resolve (swift-nio-ssh, swift-nio-transport-services)
- GhosttyKit.xcframework is built via `scripts/build_ghosttykit.sh`
- `_upstream/ghostty` is expected on branch `ios-external-backend` at commit `21c717340b62349d67124446c2447bf38796540b`
- repo-owned Ghostty fixes live in `patches/ghostty/` because `_upstream/` is gitignored
- All Geistty → iTTY symbol renames are complete
- `docs/roadmap.md` is the current implementation status reference
- `docs/engineering-standards.md` is the required quality bar for follow-on work

## Build Commands

### Daemon (Go)
```bash
cd daemon
make build        # Build binary to bin/itty
make test         # Run all tests with race detector
make lint         # Run go vet
make run          # Build and run
make cross        # Cross-compile for linux/darwin amd64/arm64
make clean        # Remove build artifacts
```

Use `make build` rather than bare `go build ./cmd/itty` so build output stays in `daemon/bin/`.

### iOS App
```bash
cd ios
xcodegen generate                    # Regenerate Xcode project from project.yml
xcodebuild -project iTTY.xcodeproj \
  -scheme iTTY \
  -destination 'generic/platform=iOS Simulator' \
  CODE_SIGNING_ALLOWED=NO \
  build                              # Verified Apple silicon simulator build (requires GhosttyKit.xcframework)
```

#### Building GhosttyKit
```bash
# Preferred path: applies repo patches, mounts Metal Toolchain if needed,
# builds GhosttyKit, installs it into ios/iTTY/Frameworks, and renames module maps.
./scripts/build_ghosttykit.sh

# Manual fallback from _upstream/ghostty:
/opt/homebrew/bin/zig build \
  -Demit-xcframework=true \
  -Dxcframework-target=universal \
  -Demit-macos-app=false
```

If `zig` on your `PATH` is not the Homebrew build, set `ZIG_BIN=/absolute/path/to/zig`
before running `scripts/build_ghosttykit.sh`.

Note: the current GhosttyKit output contains an `ios-arm64-simulator` slice, so
`ios/project.yml` intentionally excludes `x86_64` for simulator builds.

### Daemon CLI
```bash
itty              # Start daemon
itty status       # Show status (tmux, shell, sessions)
itty sessions     # List tmux sessions
itty auto on      # Enable auto-tmux shell wrapping
itty auto off     # Disable auto-tmux shell wrapping
itty version      # Print version
```

## Testing

### Daemon
```bash
cd daemon && make test
```

### Quick manual daemon test
```bash
cd daemon && make run &
curl -s localhost:8080/health | python3 -m json.tool
curl -s localhost:8080/sessions | python3 -m json.tool
kill %1
```

## Code Style

### Go (daemon)
- Standard Go formatting (`gofmt`)
- Every package has a doc comment in its primary file
- Every exported function/type has a doc comment
- Error messages: lowercase, no trailing punctuation
- Return empty slices, not nil (for JSON serialization)
- Context as first parameter on all I/O operations

## Phase 1 Discipline

Before changing Phase 1 daemon code:

1. Read `docs/roadmap.md`
2. Read `docs/engineering-standards.md`

Hard requirements:
- Do not let docs outrun code
- Do not leave public stubs on `main`
- Do not treat zero-test passes as real validation
- Do not broaden the repo shape in docs beyond what actually exists

### Swift (iOS)
- `@MainActor` on all UI-related types
- Protocols for testability (mock SSH, mock tmux surface)
- Keychain for secrets, UserDefaults for preferences
- Preserve upstream reconnect ordering and UUID-based tmux session resolution
- Session naming prefix is `itty-N` (matches daemon's auto-wrap prefix)
- `ios/project.yml` is the source of truth; regenerate with `xcodegen generate`

## Key Architecture Decisions

- **tmux is invisible plumbing** — users never interact with tmux directly
- **Desktop terminal doesn't matter** — any terminal works, tmux is the bridge
- **libghostty from day one** — Ghostty rendering, not SwiftTerm
- **Tailscale-first networking** — with manual SSH fallback
- **Connections are disposable, sessions are persistent** — iOS kills SSH, tmux keeps running
