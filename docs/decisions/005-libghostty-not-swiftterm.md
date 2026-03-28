# ADR 005: libghostty for Terminal Rendering (Not SwiftTerm)

## Status
Accepted

## Context
We need a terminal emulation engine for iOS. Two viable options:
- **SwiftTerm**: Pure Swift, proven in App Store apps, simpler integration
- **libghostty (GhosttyKit)**: Ghostty's Zig engine, Metal GPU rendering, more complex integration

## Decision
Use libghostty from day one via the Geistty fork.

## Rationale
- **User's explicit choice**: The user specified Ghostty as the terminal engine
- **GPU performance**: Metal rendering at 120fps vs SwiftTerm's CPU-based rendering
- **tmux integration**: Ghostty's viewer.zig handles tmux control mode natively — Swift never constructs tmux commands
- **Battle-tested**: libghostty powers the Ghostty desktop app used by thousands of developers
- **Proven on iOS**: Echo (commercial), VVTerm (commercial), Geistty (open source) all ship on iOS with libghostty
- **Geistty does the hard work**: The Zig↔C↔Swift bridge, external termio backend, and Metal rendering pipeline are already implemented

## Trade-offs
- Build pipeline requires Zig toolchain + XCFramework compilation
- libghostty API is not yet v1.0 — we pin versions
- Three-repo ecosystem (Geistty, Ghostty fork, SwiftNIO-SSH fork)
- More complex than SwiftTerm's pure-Swift approach

## Consequences
- We inherit Geistty's proven integration code
- Terminal rendering quality matches desktop Ghostty
- Future Ghostty improvements flow to us via upstream updates
