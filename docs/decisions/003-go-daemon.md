# ADR 003: Go for the Desktop Daemon

## Status
Accepted

## Context
The daemon needs to run on Linux and macOS (Windows stretch goal). It must be lightweight (< 10MB RSS), easy to install (single binary, no runtime), and straightforward to cross-compile.

## Decision
Write the daemon in Go.

## Rationale
- **Single binary**: No runtime dependencies. Download and run.
- **Cross-compilation**: `GOOS=darwin GOARCH=arm64 go build` — trivial.
- **Low memory**: Go binaries idle at ~5-8MB RSS. Well within our 10MB target.
- **stdlib HTTP**: `net/http` with Go 1.22+ method routing is sufficient. No framework needed.
- **Team familiarity**: Matches existing J3-Mobile Go bridge pattern.
- **Ecosystem**: gorilla/websocket for streaming, os/exec for tmux, sideshow/apns2 for push.

## Alternatives Considered
- **Rust**: Better memory guarantees but longer development time, more complex cross-compilation with platform-specific code (D-Bus, AppleScript).
- **Python**: Too heavy, requires runtime installation.
- **Shell script**: Not powerful enough for REST API + WebSocket.

## Consequences
- Go's goroutine model maps well to concurrent session monitoring
- Platform-specific code uses build tags (`windows_linux.go`, `windows_darwin.go`)
- Binary size is ~6-7MB (acceptable for a desktop daemon)
