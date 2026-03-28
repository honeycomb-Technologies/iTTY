# ADR 002: tmux as the Persistence Layer

## Status
Accepted

## Context
iOS kills background apps after ~3 minutes. SSH connections die when the app is backgrounded or killed. We need sessions to persist across app closures.

## Decision
Use tmux on the desktop as invisible plumbing for session persistence. The iTTY daemon auto-configures the user's shell to wrap new terminals in tmux sessions.

## Why tmux (not a custom solution)
- tmux is battle-tested (15+ years, installed on most Linux/macOS machines)
- Multiple clients can attach to the same session simultaneously (phone + desktop see same state)
- Control mode (-CC) provides machine-readable structured output — already integrated into Ghostty's viewer.zig
- Sessions persist indefinitely on the server regardless of client state
- Building our own session multiplexer would be reimplementing tmux

## Why invisible
Users should never type `tmux` or know it exists. The daemon:
1. Auto-installs tmux if missing
2. Auto-configures the shell rc file to wrap new terminals in tmux
3. Guards against nesting, VS Code, Emacs, and user opt-out

## Consequences
- Desktop requires tmux installed (daemon handles this)
- Auto-wrap modifies user's .bashrc/.zshrc (with clear markers, idempotent, removable)
- Existing processes not in tmux require best-effort migration (reptyr)
- Any desktop terminal works — tmux is terminal-agnostic
