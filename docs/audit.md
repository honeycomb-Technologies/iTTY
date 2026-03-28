# Geistty Codebase Audit

**Date**: 2026-03-28
**Source**: github.com/daiimus/geistty (v0.1-stable)
**Scope**: All 49 Swift files, ARCHITECTURE.md, build system

## Summary

Geistty is a well-engineered proof-of-concept with solid foundations. The terminal engine integration (libghostty), SSH transport, and tmux control mode are production-quality. The app layer needs refactoring for maintainability and App Store readiness. **Recommendation: adopt as iTTY foundation — fork, don't rewrite.**

---

## Codebase Statistics

| Metric | Value |
|--------|-------|
| Swift files | 49 |
| Test functions | ~920 |
| Lines of Swift | ~14,786 (tests) + ~8,000 (source) |
| Test files | 24 unit + 4 UI |
| Critical files (>1000 lines) | 4 |

## The Four Critical Files

### 1. Ghostty.swift (2,837 lines) — SurfaceView

The UIView subclass that hosts Metal rendering and bridges Swift ↔ Ghostty Zig.

**Strengths**:
- Callback isolation: C callbacks from Zig dispatch via `DispatchQueue.main` (not `@MainActor`) — this is correct because C callbacks can't use Swift concurrency
- Conservative cleanup: explicit `close()` with `deinit` assertions
- CADisplayLink retain cycle prevention
- 90+ C API calls properly wrapped

**Issues**:
- Magic number `14.0` for default font size — should be configurable
- Scroll margins hardcoded
- Massive file — could extract input handling, search, tmux queries into extensions

**Preserve for iTTY**: The entire C API bridge layer. Do not rewrite — extend.

### 2. SSHSession.swift (1,749 lines) — Connection Lifecycle

Manages SSH connections, data routing, and reconnection.

**Strengths**:
- AsyncStream for write queuing — proper async/await integration
- Early receive buffer: stores SSH data arriving before delegate is set
- Background task protection during app suspension handshake
- 3-retry reconnection with stored credentials

**Issues**:
- **Issue #3 root cause (background reconnection)**: Upstream already has app-lifecycle handling across `ContentView`, `TerminalContainerView`, and `SSHSession`, but the ownership is spread across layers and is hard to reason about. iTTY should extract a dedicated connection/reconnect coordinator without regressing the existing synchronous ordering.
- **Issue #4 migration risk (session name tracking)**: Upstream already uses a UUID-derived sentinel in `TmuxSessionNameResolver` and has regression tests for shell-echo false positives. The real risk in iTTY is losing that behavior during the move, not inventing it from scratch.
- Unbounded write queue buffer — no backpressure, potential OOM under sustained load
- No exponential backoff on reconnection (fixed 5s delay)
- SSH write failures don't stop the rendering pipeline

**Preserve for iTTY**: The SSH handshake, auth, data routing, and current reconnect sequencing. If refactored, extract it into an explicit connection manager instead of rewriting behavior blind.

### 3. TmuxSessionManager.swift (2,062 lines) — State Hub

Central tmux state management: windows, panes, surfaces.

**Strengths**:
- Sophisticated surface promotion when primary pane closes
- Clean notification handling from Ghostty's Zig viewer
- Layout parsing via TmuxSplitTree (binary tree of splits)
- Surface-per-pane model with proper lifecycle

**Issues**:
- **Surface ownership is fragile**: three-way model (primary + observers + viewerOwner) with stashing pattern. A dedicated `TmuxSurfaceRegistry` struct would be cleaner.
- **4 potential race conditions identified** with mitigations documented but not all enforced
- `pendingOutput` and `pendingCommands` are unbounded queues — OOM risk
- High complexity: 2,062 lines with interleaved state management

**Preserve for iTTY**: The tmux state model and notification handling. Extract surface lifecycle into a dedicated registry.

### 4. TerminalContainerView.swift (1,145 lines) — UI Bridge

SwiftUI ↔ UIKit bridge hosting the terminal.

**Strengths**:
- Two-layer architecture (SwiftUI + UIKit) prevents gray-flash on navigation
- Idempotent lifecycle: Setup → Running → Disconnect → Teardown
- Pre-surface buffering mirrors SSHSession pattern (consistency)
- Proper focus/visibility management

**Issues**:
- Monolithic ViewModel — should split connection state from UI state
- No error handling for surface creation failure (blank screen)
- No surface creation timeout

**Preserve for iTTY**: The UIKit bridge pattern. Add error views and timeout handling.

---

## Supporting Files

### NIOSSHConnection.swift (787 lines)
- **Strength**: Sophisticated timeout handling with task group race for SSH handshake
- **Strength**: TOFU host key verification (fail-closed on all errors)
- **Strength**: Network path monitoring (WiFi ↔ cellular detection)
- **Issue**: Optimistic network recovery — stays stale until data arrives
- **Issue**: PTY terminal type "xterm-256color" hardcoded

### ConnectionProfile.swift (442 lines)
- **Production-ready**: iCloud sync with tombstone-tracked deletes
- **Clean migration**: Old profiles gain tmux/Files fields automatically
- **Direct reuse**: Can be adopted into iTTY with minimal changes

### Test Suite (~920 functions)
- **Excellent tmux coverage**: 11 test files covering control mode, layout, state reconciliation
- **Good mock patterns**: `MockSSHSessionDelegate`, `MockTmuxSurface` via protocols
- **Gap**: No integration tests for full SSH → tmux → render pipeline
- **Gap**: No memory/performance benchmarks

---

## Issue Root Causes

### Issue #3: Background Reconnection
**Symptom**: Background/foreground reconnect logic is hard to audit and easy to break during refactoring.
**Root cause**: Upstream already uses app-layer lifecycle hooks (`scenePhase` gating in `ContentView`, app-active notifications in `TerminalContainerView`, and reconnect methods in `SSHSession`), but the behavior is spread across multiple layers instead of being owned by one coordinator.
**Fix**: Extract a dedicated `ConnectionManager` or `AutoReconnectService` that preserves the current sequencing while making lifecycle ownership explicit. The goal is consolidation, not replacing upstream behavior with a brand-new reconnect model.

### Issue #4: tmux Session Name Tracking
**Symptom**: App attaches to wrong tmux session.
**Root cause**: The unsafe version of this logic was a fixed-string sentinel. Upstream has already corrected it by using a UUID-derived sentinel per resolution attempt and by testing shell-echo edge cases.
**Fix**: Preserve the current `TmuxSessionNameResolver` UUID-sentinel behavior and its tests during migration. Do not reintroduce fixed-string or substring-based matching.

---

## Refactoring Targets

### What Moves Where (Geistty → iTTY module mapping)

Paths below are relative to `ios/iTTY/Sources/`.

```
GEISTTY FILE                        → iTTY LOCATION                    ACTION
─────────────────────────────────────────────────────────────────────────────
App/GeisttyApp.swift                → App/iTTYApp.swift                Rename + extend
App/ContentView.swift               → App/ContentView.swift            Replace (new nav)

Auth/ConnectionProfile.swift        → Core/Auth/ConnectionProfile.swift Keep
Auth/CredentialProvider.swift        → Core/Auth/CredentialProvider.swift Keep
Auth/KeychainManager.swift           → Core/Auth/KeychainManager.swift   Keep
Auth/SSHKeyManager.swift             → Core/Auth/SSHKeyManager.swift     Keep
Auth/SSHKeyParser.swift              → Core/Auth/SSHKeyParser.swift      Keep
Auth/BiometricGatekeeper.swift       → Core/Auth/BiometricGatekeeper.swift Keep

Ghostty/Ghostty.swift               → Core/Terminal/SurfaceView.swift   Refactor (extract)
Ghostty/Ghostty.App.swift            → Core/Terminal/GhosttyRuntime.swift Rename
Ghostty/Ghostty.Config.swift         → Core/Terminal/GhosttyConfig.swift  Rename
Ghostty/Ghostty.Command.swift        → Core/Terminal/Commands.swift      Rename
Ghostty/Ghostty.SearchState.swift    → Core/Terminal/SearchState.swift   Rename
Ghostty/GhosttyInput.swift           → Core/Terminal/InputTranslation.swift Rename
Ghostty/FontMapping.swift            → Core/Config/FontMapping.swift     Move
Ghostty/ConfigSyncManager.swift      → Core/Config/ConfigSyncManager.swift Move
Ghostty/SurfaceSearchOverlay.swift   → Core/Terminal/SearchOverlay.swift  Rename
Ghostty/SelectionOverlay.swift       → Core/Terminal/SelectionOverlay.swift Rename
Ghostty/TmuxSurfaceProtocol.swift    → Core/Tmux/TmuxSurfaceProtocol.swift Move
Ghostty/Ghostty.SurfaceConfiguration.swift → Core/Terminal/SurfaceConfiguration.swift Rename

SSH/SSHSession.swift                 → Core/SSH/SSHSession.swift         Refactor
SSH/NIOSSHConnection.swift           → Core/SSH/NIOSSHConnection.swift   Keep
SSH/SSHCommandRunner.swift           → Core/SSH/SSHCommandRunner.swift   Keep
SSH/TmuxSessionManager.swift         → Core/Tmux/TmuxSessionManager.swift Refactor (extract registry)
SSH/TmuxLayout.swift                 → Core/Tmux/TmuxLayout.swift       Keep
SSH/TmuxSplitTree.swift              → Core/Tmux/TmuxSplitTree.swift    Keep
SSH/TmuxModels.swift                 → Core/Tmux/TmuxModels.swift       Keep
SSH/TmuxSessionNameResolver.swift    → Core/Tmux/TmuxSessionNameResolver.swift Keep (preserve UUID sentinel)
SSH/TmuxWireDiagnostics.swift        → Core/Tmux/TmuxWireDiagnostics.swift Keep

Terminal/TerminalContainerView.swift  → Features/Terminal/TerminalContainerView.swift Refactor
Terminal/RawTerminalUIViewController+Keyboard.swift → Features/Terminal/Keyboard.swift Rename
Terminal/RawTerminalUIViewController+MenuBar.swift  → Features/Terminal/MenuBar.swift  Rename
Terminal/RawTerminalUIViewController+Search.swift   → Features/Terminal/Search.swift   Rename
Terminal/RawTerminalUIViewController+Shortcuts.swift → Features/Terminal/Shortcuts.swift Rename
Terminal/RawTerminalUIViewController+Tmux.swift     → Features/Terminal/TmuxPane.swift  Rename
Terminal/RawTerminalUIViewController+StatusBar.swift → Features/Terminal/StatusBar.swift Rename
Terminal/RawTerminalUIViewController+WindowPicker.swift → Features/Terminal/WindowPicker.swift Rename
Terminal/Theme.swift                  → Core/Config/ThemeManager.swift    Move
Terminal/TmuxMultiPaneView.swift      → Features/Terminal/MultiPaneView.swift Rename
Terminal/TmuxSplitView.swift          → Features/Terminal/SplitView.swift  Rename
Terminal/TmuxWindowPickerView.swift   → Features/Terminal/WindowPickerView.swift Rename
Terminal/TmuxSessionPickerView.swift  → Features/Terminal/SessionPickerView.swift Rename
Terminal/TmuxStatusBarView.swift      → Features/Terminal/StatusBarView.swift Rename
Terminal/CommandPaletteView.swift     → Features/Terminal/CommandPalette.swift Rename
Terminal/TerminalToolbar.swift        → Features/Terminal/Toolbar.swift    Rename

UI/ConnectionListView.swift          → Features/Machines/ConnectionListView.swift Move (evolves into MachineListView)
UI/ConnectionEditorView.swift        → Features/Machines/ConnectionEditorView.swift Move (evolves into AddMachineView)
UI/SettingsView.swift                → Features/Settings/SettingsView.swift Move
UI/KeyTableIndicatorView.swift       → Features/Terminal/KeyTableIndicator.swift Move

(NEW FILES — not in Geistty)
                                     → Features/SessionBrowser/SessionBrowserView.swift
                                     → Features/SessionBrowser/SessionRowView.swift
                                     → Features/Machines/MachineListView.swift
                                     → Features/Machines/AddMachineView.swift
                                     → Features/Machines/TailscaleDiscoveryView.swift
                                     → Features/Terminal/TerminalTabBar.swift
                                     → Features/Onboarding/OnboardingView.swift
                                     → Services/Tailscale/TailscaleAPIClient.swift
                                     → Services/Tailscale/TailscaleVPNDetector.swift
                                     → Services/Daemon/DaemonClient.swift
                                     → Services/Connection/ConnectionManager.swift
                                     → Services/Connection/AutoReconnectService.swift
                                     → Models/Machine.swift
                                     → Models/SavedSession.swift
```

### Priority Refactoring (Phase 2 scope)

**P0 — Correctness** (do first):
1. Extract reconnect ownership into `ConnectionManager` or `AutoReconnectService` without changing the existing upstream lifecycle sequencing
2. Add bounded queues with shedding in `SSHSession` and `TmuxSessionManager`
3. Preserve the UUID-based `TmuxSessionNameResolver` behavior and carry its regression tests forward during the migration

**P1 — Structure** (do second):
4. Extract `TmuxSurfaceRegistry` from `TmuxSessionManager`
5. Split `TerminalContainerView` ViewModel into connection state + UI state
6. Reorganize into `Core/` and `Features/` directory structure and wire the real Xcode targets around it

**P2 — Product Integration** (do third):
7. Add `DaemonClient` and session browser flows against the current daemon API
8. Add surface creation timeout + error view
9. Add exponential backoff to SSHSession reconnection plus SSH write failure detection → UI error state

---

## What Requires macOS

The following audit items can only be verified/implemented on macOS with Xcode:
- Xcode target creation, project wiring, and build setting migration
- Metal rendering verification after refactoring
- Instruments memory profiling
- Device testing for reconnection fixes
- UI test updates for new module structure

The source/resource/test scaffold can be prepared on Linux, but the final Phase 2 implementation still requires macOS for project wiring, build validation, and device testing.
