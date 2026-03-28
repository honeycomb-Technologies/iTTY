# iTTY — Development Guide

## Project Overview

iTTY is a two-component persistent remote terminal system:
- **iOS App** (`/ios/`): Ghostty-powered terminal (Geistty fork, libghostty Metal rendering)
- **Desktop Daemon** (`/daemon/`): Go binary that manages tmux sessions and exposes them via REST API

## Repository Structure

```
iTTY/
├── ios/          # iOS app (Geistty fork) — Swift/SwiftUI + GhosttyKit
├── daemon/       # Desktop daemon — Go
├── docs/         # All documentation
├── install/      # Install scripts (systemd, launchd)
└── _upstream/    # Upstream Geistty clone (reference only, not part of build)
```

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

### iOS App
```bash
# Requires macOS with Xcode 16+ and Zig 0.14+
# See docs/ios/building.md for full instructions
cd ios
# Build GhosttyKit first (from Ghostty fork)
# Then: xcodebuild -scheme iTTY -destination 'platform=iOS'
```

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

### Swift (iOS)
- Follow existing Geistty patterns
- `@MainActor` on all UI-related types
- Protocols for testability (mock SSH, mock tmux surface)
- Core Data for persistence, Keychain for secrets

## Key Architecture Decisions

- **tmux is invisible plumbing** — users never interact with tmux directly
- **Desktop terminal doesn't matter** — any terminal works, tmux is the bridge
- **libghostty from day one** — Ghostty rendering, not SwiftTerm
- **Tailscale-first networking** — with manual SSH fallback
- **Connections are disposable, sessions are persistent** — iOS kills SSH, tmux keeps running
