# iTTY — Development Guide

## Project Overview

iTTY targets a two-component persistent remote terminal system:
- **Desktop Daemon** (`/daemon/`): Go binary that manages tmux sessions and exposes them via REST API
- **iOS App**: planned integration based on the Geistty fork and libghostty

## Repository Structure

```
iTTY/
├── daemon/       # Desktop daemon — Go
├── ios/          # iOS scaffold copied from upstream Geistty; not buildable yet
├── docs/         # Documentation, roadmap, and engineering standards
└── _upstream/    # Upstream Geistty clone (reference only, not part of build)
```

Current reality:
- `ios/` now contains a Linux-generated source/resource/test scaffold under `ios/iTTY/`
- there is still no buildable Xcode target in this repository
- `_upstream/geistty/` remains the upstream source of truth for project wiring and parity checks
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
make scaffold-ios       # Copy the verified scaffold into ios/
make check-ios-scaffold # Verify scaffold + manifest completeness

# Build and runtime verification still require macOS + Xcode.
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
- Follow existing Geistty patterns
- `@MainActor` on all UI-related types
- Protocols for testability (mock SSH, mock tmux surface)
- Core Data for persistence, Keychain for secrets
- Preserve upstream reconnect ordering and UUID-based tmux session resolution during migration

## Key Architecture Decisions

- **tmux is invisible plumbing** — users never interact with tmux directly
- **Desktop terminal doesn't matter** — any terminal works, tmux is the bridge
- **libghostty from day one** — Ghostty rendering, not SwiftTerm
- **Tailscale-first networking** — with manual SSH fallback
- **Connections are disposable, sessions are persistent** — iOS kills SSH, tmux keeps running
