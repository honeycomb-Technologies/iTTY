# iTTY Documentation

## Overview
- [Architecture](architecture.md) — Full system design (iOS app + daemon)
- [Development](development.md) — How to build, test, and contribute
- [Roadmap](roadmap.md) — Implementation phases and milestones

## iOS App
- [Building](ios/building.md) — Build from source
- [Project Structure](ios/project-structure.md) — Every directory and file explained
- [libghostty Integration](ios/libghostty-integration.md) — How GhosttyKit is built and used
- [Metal Rendering](ios/metal-rendering.md) — GPU rendering pipeline
- [SSH Implementation](ios/ssh-implementation.md) — SwiftNIO-SSH fork details
- [tmux Control Mode](ios/tmux-control-mode.md) — How -CC integration works

## Desktop Daemon
- [Architecture](daemon/architecture.md) — Daemon internals
- [API Reference](daemon/api.md) — REST + WebSocket endpoints
- [Installation](daemon/install.md) — Setup on Linux, macOS, Windows
- [Platforms](daemon/platforms.md) — Platform-specific behavior
- [Configuration](daemon/configuration.md) — Config file reference

## Features
- [Auto-Reconnect](features/auto-reconnect.md) — Background/foreground reconnection
- [Machines](features/machines.md) — Machine management
- [Sessions](features/sessions.md) — Session browser
- [Tabs](features/tabs.md) — Multi-tab system
- [Tailscale](features/tailscale.md) — Tailscale integration
- [Daemon Integration](features/daemon-integration.md) — App ↔ daemon communication
- [Notifications](features/notifications.md) — Push notifications
- [Mosh](features/mosh.md) — Mosh support

## Decisions
- [001: Geistty Fork](decisions/001-geistty-fork.md) — Why we forked Geistty
- [002: tmux as Persistence](decisions/002-tmux-as-persistence.md) — Why tmux, not a custom solution
- [003: Go Daemon](decisions/003-go-daemon.md) — Why Go for the daemon
- [004: Tailscale First](decisions/004-tailscale-first.md) — Why Tailscale-first networking
- [005: libghostty](decisions/005-libghostty-not-swiftterm.md) — Why Ghostty rendering
