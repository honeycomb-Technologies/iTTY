# Codex Handoff — Phase 2 iOS App

Last updated: 2026-03-29

## Your Role

You are auditing, refining, and completing the Phase 2 iOS work. The previous session set up the project infrastructure. Your job is to take it to 10/10.

## Current State

### What's Done
- Xcode project generated via xcodegen (`ios/project.yml` → `ios/iTTY.xcodeproj`)
- Three targets: iTTY (app), iTTYTests, iTTYUITests
- SPM dependencies resolve: swift-nio-ssh (daiimus fork, `add-rsa-support` branch), swift-nio-transport-services
- Full Geistty → iTTY rename complete (0 Geistty references in Swift files, 661 Ghostty/GhosttyKit references preserved)
- Info.plist with Face ID, background processing, Bonjour SSH discovery, 15 custom fonts
- Entitlements with Keychain sharing (`com.itty.shared`) and app groups
- Zig upgraded to 0.15.2

### What Blocks Compilation
**GhosttyKit.xcframework** is missing. This is the Ghostty terminal engine compiled as a static library for iOS. Without it, ~15 source files that `import GhosttyKit` fail.

The framework is built from `github.com/daiimus/ghostty` (branch `ios-external-backend`) using:
```bash
zig build -Demit-xcframework=true -Dxcframework-target=universal
```

Current blocker: The Zig cross-compilation produces linker errors for standard C symbols (`_fork`, `_malloc_size`, `_dispatch_time`, etc.) when targeting iOS. This is likely a missing `-lSystem` or incorrect SDK path in the Zig build configuration.

**To investigate:**
1. Check if `_upstream/ghostty/build.zig` has iOS-specific target options
2. The upstream `ci.sh` script may set environment variables needed for cross-compilation
3. Try specifying the iOS SDK path explicitly: `-Dsysroot=$(xcrun --sdk iphonesimulator --show-sdk-path)`
4. Check if the build needs macOS host + iOS cross targets specified separately

### What Needs Fixing in project.yml Once GhosttyKit Is Available
In `ios/project.yml`, uncomment:
- The framework dependency line
- The `OTHER_SWIFT_FLAGS` module map flags
- The `SWIFT_INCLUDE_PATHS` for GhosttyKit headers
- The per-config (Debug/Release) overrides for simulator vs device paths

Then regenerate: `cd ios && xcodegen generate`

## Remaining Phase 2 Work

### Priority 1: Get GhosttyKit Building
Debug the Zig cross-compilation. See `docs/ios/building.md` for the full build steps.

### Priority 2: Fix Compilation Errors
Once GhosttyKit is available, build the app and fix errors in waves:
1. GhosttyKit-dependent type resolution
2. Swift 5.9+ compatibility (code was written for Swift 5.x, Xcode uses 5.0 setting)
3. Any remaining symbol/type mismatches from the Geistty → iTTY rename

### Priority 3: Implement Missing Services
From `ios/phase2-manifest.json` `planned_new_files`:
- `Services/Connection/ConnectionManager.swift` — extract reconnect ownership from SSHSession
- `Services/Connection/AutoReconnectService.swift` — background→foreground reconnect logic
- `Services/Tailscale/TailscaleDiscoveryService.swift` — Tailscale device enumeration
- `Features/Machines/TailscaleDiscoveryView.swift` — UI for discovered Tailscale devices
- `Features/Terminal/TerminalTabBar.swift` — multi-tab terminal management
- `Features/Onboarding/OnboardingView.swift` — first-run experience

### Priority 4: Test Strategy
Decide for each test file in `ios/iTTYTests/UpstreamParity/`:
- **Keep as-is**: Tests that validate logic unchanged from upstream (tmux parsing, key management)
- **Adapt**: Tests that reference renamed identifiers (already done) but may need behavior updates
- **Replace**: Tests for features being redesigned (reconnect, session browser)

### Priority 5: Device Verification
- Metal rendering on simulator and device
- Background → foreground SSH reconnect
- tmux multi-pane state survival across reconnect
- Surface ownership stability under pane operations
- Daemon session browsing and attach flow

## Key Files to Know

| File | Why It Matters |
|------|---------------|
| `ios/project.yml` | Source of truth for Xcode project (xcodegen) |
| `ios/iTTY/Info.plist` | App capabilities and permissions |
| `ios/iTTY/iTTY.entitlements` | Keychain and app group config |
| `ios/iTTY/Sources/Core/SSH/SSHSession.swift` | Connection lifecycle, tmux integration |
| `ios/iTTY/Sources/Core/Tmux/TmuxSessionManager.swift` | Central tmux state hub (2000+ lines) |
| `ios/iTTY/Sources/Core/Tmux/TmuxSessionNameResolver.swift` | `itty-N` session naming logic |
| `ios/iTTY/Sources/Core/Auth/KeychainManager.swift` | Keychain with `com.itty` identifiers |
| `ios/iTTY/Sources/Services/Daemon/DaemonClient.swift` | HTTP client for daemon communication |
| `ios/phase2-manifest.json` | Source mapping and planned files list |
| `docs/audit.md` | Upstream code audit with issues #3 and #4 |
| `docs/engineering-standards.md` | Quality bar for all changes |

## Engineering Standards (Non-Negotiable)

From `docs/engineering-standards.md`:
- Code, docs, and shipped behavior must agree in the same change
- Public API work requires explicit tests for success and failure cases
- A feature is not considered implemented until it is verified end to end
- Small, finished vertical slices over broad speculative scaffolding
- Do not document things that do not exist yet without marking them as planned

## Known Technical Debt (from docs/audit.md)

- **Unbounded queues** in SSHSession and TmuxSessionManager (OOM risk)
- **Fragile surface ownership** — three-way model in TmuxSessionManager
- **Spread reconnect logic** across ContentView, TerminalContainerView, SSHSession
- **No timeout** on surface creation in TerminalContainerView
- **Hardcoded values** in SurfaceView (font size 14.0, scroll margins)

## Build Verification

After your changes, verify:
```bash
# Daemon still passes
cd daemon && make test && make lint

# iOS project regenerates cleanly
cd ios && xcodegen generate

# iOS builds (once GhosttyKit is available)
xcodebuild -project ios/iTTY.xcodeproj -scheme iTTY \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build

# Rename is clean
grep -rn "Geistty\|geistty\|GEISTTY" ios/ --include="*.swift" | grep -v phase2-manifest | wc -l
# Must be 0
```
