# Daemon Platforms

## Supported In Phase 1

- Linux
- macOS

## Linux

- shell auto-wrap supports bash, zsh, and fish
- window discovery uses `wmctrl` when available
- if `wmctrl` is missing, `GET /windows` returns an empty list

## macOS

- shell auto-wrap supports bash, zsh, and fish
- window discovery uses AppleScript through `osascript`
- if window inspection fails, the daemon returns an empty list

## Windows

Windows is not part of the Phase 1 build matrix.

## Cross-Compilation

The daemon Makefile cross-compiles for:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
