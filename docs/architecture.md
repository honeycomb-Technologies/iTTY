# iTTY Architecture

This document describes the target system shape. For the current implementation boundary, use `docs/roadmap.md`.

## System Overview

iTTY consists of two components that communicate over a Tailscale private network:

```
┌─────────────────────────────────────────┐
│   Desktop (Linux/macOS)                 │
│                                         │
│  Any Terminal ──→ tmux (auto-wrapped)   │
│                     │                   │
│        ┌────────────┴──────────────┐    │
│        │  iTTY Daemon (Go)         │    │
│        │  • Auto-installs tmux     │    │
│        │  • Auto-configures shell  │    │
│        │  • Reports tmux sessions  │    │
│        │  • REST API today         │    │
│        │  • WebSocket/APNs later   │    │
│        └───────────┬───────────────┘    │
│              Tailscale Serve :8080      │
└────────────────────┬────────────────────┘
                     │ (private tailnet)
┌────────────────────▼────────────────────┐
│   iPhone                                │
│                                         │
│  iTTY App (Geistty fork + libghostty)   │
│  • Phase 2 scaffold in-repo             │
│  • Daemon browser and attach planned    │
│  • Metal terminal via Geistty foundation│
│  • Reconnect refinement planned         │
│  • Multi-tab polish later               │
└─────────────────────────────────────────┘
```

## iOS App Architecture

The iOS app is forked from [Geistty](https://github.com/daiimus/geistty) (MIT license), which provides the hard-won integration between Swift and Ghostty's Zig terminal engine.

### Layer Diagram

```
SwiftUI (navigation, settings, session browser)
    ↓
UIKit Bridge (SurfaceView — Metal rendering + keyboard input)
    ↓
GhosttyKit (Zig/C — VT parsing, terminal state, Metal renderer)
    ↓
External termio backend (receives SSH bytes, not local PTY)
    ↓
SwiftNIO-SSH (async networking)
```

### Data Flow: User Types → Screen Updates

**Output path** (command results appear on screen):
```
SSH server sends response bytes
  → NIOSSHConnection receives on NIO event loop
  → SSHSession.handleReceivedData()
  → ghostty_surface_write_output() [C API call into Zig]
  → VT parser processes escape sequences
  → Terminal grid updated (cursor, styles, text)
  → Metal renderer draws changed cells at 120fps
  → SurfaceView displays on screen
```

**Input path** (user types a key):
```
User taps key on iPhone keyboard
  → SurfaceView.insertText() [UIKeyInput]
  → ghostty_surface_text() [C API call into Zig]
  → Zig encodes key (Kitty keyboard protocol)
  → write_callback fires [C callback from Zig → Swift]
  → SurfaceView.onWrite?(data)
  → SSHSession.sendInput(data)
  → NIOSSHConnection.writeAsync(data)
  → SSH sends bytes to server
```

### tmux Control Mode

When connected to a tmux session, Ghostty's `viewer.zig` transparently wraps all I/O:

**User input becomes tmux commands:**
```
User types "ls\n"
  → Ghostty's viewer.zig wraps it:
    "send-keys -H -t %42 6C 73 0D\n"
  → Sent to tmux on server via SSH
```

**Server output is parsed from tmux protocol:**
```
tmux sends: "%output %42 \033[1mhello\033[m"
  → viewer.zig extracts pane ID (42) and data
  → Dispatches to correct pane's Terminal instance
  → Metal renders that pane
```

**Swift never constructs tmux commands.** It's a pure byte pass-through. All protocol handling lives in Ghostty's Zig code.

## Desktop Daemon Architecture

The daemon is a single Go binary with no runtime dependencies (beyond tmux).

### Module Structure

```
daemon/
├── cmd/itty/main.go        — CLI entry point
├── internal/
│   ├── api/                 — HTTP + WebSocket server
│   ├── tmux/                — tmux command execution + parsing
│   ├── shell/               — Shell detection + auto-configuration
│   ├── platform/            — OS-specific window discovery
│   ├── notify/              — Apple Push Notification Service
│   └── config/              — Configuration management
```

### How Auto-Wrap Works

When the daemon is installed, it adds a guarded snippet to the user's shell rc file:

```bash
# >>> iTTY auto-session >>>
if [ -z "$TMUX" ] && [ -z "$ITTY_NOAUTO" ] && ...; then
  exec tmux new-session -A -s "itty-<tty-name>"
fi
# <<< iTTY auto-session <<<
```

**Guards prevent**:
- Nesting (already inside tmux)
- User opt-out (`ITTY_NOAUTO=1`)
- VS Code integrated terminal
- Emacs shell

Every new terminal window automatically gets a tmux session. The user never sees tmux — it just works.

### Session Persistence Model

```
Phone opens session:
  iTTY → SSH → creates tmux session "itty-pts-3"
  → User works (Claude Code, vim, etc.)
  → iOS kills app → SSH dies → tmux keeps running
  → User reopens app → SSH reconnect → tmux attach
  → Same session, same state, ~1-2 seconds

Desktop has sessions:
  Auto-wrapped terminal → tmux session "itty-pts-1"
  → Daemon reports session via REST API
  → Phone app sees it → user taps → SSH + tmux attach
  → Both phone and desktop see same session simultaneously
```

## Key Design Principles

1. **tmux is invisible** — users never type tmux commands
2. **Desktop terminal is irrelevant** — any terminal works
3. **Connections are disposable** — SSH dies, tmux persists
4. **Zero config** — install daemon, install app, done
5. **Low memory** — daemon < 10MB, app carefully profiled
