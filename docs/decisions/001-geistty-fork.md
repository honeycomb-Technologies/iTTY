# ADR 001: Fork Geistty as iOS App Foundation

## Status
Accepted

## Context
We need an iOS terminal app powered by Ghostty's libghostty engine. Building the libghostty integration from scratch would require:
- Zig → C → Swift bridge implementation
- Metal rendering pipeline for iOS
- External termio backend (iOS has no PTY/fork/exec)
- tmux control mode (-CC) parsing
- SSH integration with the terminal engine

## Decision
Fork [Geistty](https://github.com/daiimus/geistty) (MIT license), which already solves all of the above.

## What Geistty Provides
- 49 Swift files across 6 modules
- ~920 test functions
- GhosttyKit XCFramework with Metal GPU rendering at 120fps
- External termio backend (SSH bytes in, rendered output out)
- Native tmux -CC control mode via Ghostty's viewer.zig
- SSH via SwiftNIO-SSH fork (RSA + Ed25519 + ECDSA)
- iOS Keychain integration for SSH keys

## What We Change
- **Refactor**: proof-of-concept architecture → clean module separation
- **Fix bugs**: background reconnection (#3), session name tracking (#4)
- **Add features**: Tailscale discovery, session browser, machine management, multi-tab, daemon communication, push notifications, auto-reconnect

## Consequences
- We inherit Geistty's three-repo ecosystem (main app, Ghostty fork, SwiftNIO-SSH fork)
- We must track upstream changes and cherry-pick relevant fixes
- Our additions go in new files to minimize merge conflicts
- We take on the responsibility of bringing the codebase to App Store quality
