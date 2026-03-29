# Codex Session Prompt

Copy everything below the line and paste as the opening prompt for a fresh Codex session.

---

You are continuing Phase 2 work on the iTTY iOS app. Read these files first to get full context:

1. `docs/ios/codex-handoff.md` — your primary briefing (current state, what's done, what remains)
2. `docs/roadmap.md` — overall project status
3. `docs/engineering-standards.md` — quality bar (non-negotiable)
4. `docs/audit.md` — upstream code audit with known issues
5. `CLAUDE.md` — build commands and project conventions
6. `ios/project.yml` — Xcode project configuration (source of truth)

## Your Mission

1. **Build GhosttyKit.xcframework** — the Zig cross-compilation from `_upstream/ghostty` (branch `ios-external-backend`) has linker errors for standard C symbols when targeting iOS. Debug the build system. Zig 0.15.2 is installed. The build command is `zig build -Demit-xcframework=true -Dxcframework-target=universal`. Check `build.zig` for iOS-specific target options and SDK path configuration.

2. **Get the iOS app compiling** — once GhosttyKit is available, uncomment the framework dependency and module map flags in `ios/project.yml`, regenerate with `xcodegen generate`, and fix any Swift compilation errors.

3. **Audit the Geistty → iTTY rename** — verify the rename is complete and correct. Check that session naming (`itty-N`) aligns with the daemon's auto-wrap prefix in `daemon/internal/shell/configure.go`. Check Keychain identifiers (`com.itty`) are consistent.

4. **Audit project configuration** — verify `ios/project.yml` settings match upstream Geistty (deployment target, Swift version, framework linking, SPM versions). Cross-reference with `_upstream/geistty/Geistty/Geistty.xcodeproj/project.pbxproj`.

5. **Implement missing services** if time permits — see `ios/phase2-manifest.json` `planned_new_files` for the list (ConnectionManager, AutoReconnectService, TailscaleDiscovery).

6. **Update docs** to reflect your changes — `docs/roadmap.md`, `docs/ios/building.md`, and `CLAUDE.md` must stay synchronized with reality.

## Rules

- Read `docs/engineering-standards.md` before writing any code
- Do not let docs outrun code
- Run `cd daemon && make test && make lint` to verify you haven't broken the daemon
- Keep commits small and focused
- The session naming prefix `itty-` in the iOS app must match the daemon's `itty-` prefix in `daemon/internal/shell/configure.go`
