# Phase 2 Handoff

## Verified Corrections

- Issue #3 in `docs/audit.md` was too strong. Upstream already has app-lifecycle reconnect handling spread across `ContentView.swift`, `TerminalContainerView.swift`, and `SSHSession.swift`. The Phase 2 job is to extract that behavior into clearer ownership without losing the current synchronous ordering.
- Issue #4 in `docs/audit.md` was stale. Upstream already uses a UUID-based sentinel in `TmuxSessionNameResolver.swift`, and the parity tests cover the false-positive cases. The migration goal is to preserve that behavior, not re-invent it.
- The source map is now checked against the real upstream tree. All 49 Swift source files are accounted for, and the `RawTerminalUIViewController+...` path typos have been corrected.

## Linux-Safe Work Completed

- `scripts/phase2_scaffold.py` validates the upstream source map, exports `ios/phase2-manifest.json`, and copies the current scaffold into `ios/`.
- `ios/iTTY/Sources/` now contains the mapped Swift source scaffold.
- `ios/iTTY/Assets.xcassets` and `ios/iTTY/Resources` now contain the upstream assets and resources required for later project wiring.
- `ios/iTTYTests/UpstreamParity/` and `ios/iTTYUITests/UpstreamParity/` now contain the upstream test files for later parity decisions on macOS.
- The first iTTY-owned daemon browse layer now exists in source: machine models/store, daemon client, machine list, add-machine flow, session browser, linked-profile attach handoff, and daemon JSON tests.

## Mac-Only Remaining Work

1. Create the real iTTY Xcode project or retarget the upstream project into this repo.
2. Wire the scaffolded files, assets, and resources into app/test/UI-test targets.
3. Resolve compile-time symbol/module renames where files still contain upstream Geistty identifiers.
4. Wire the new daemon browser/machine files into the real app navigation and test targets where the Linux pass could only do source-level integration.
5. Implement the still-missing iTTY-specific files: onboarding, Tailscale discovery/client, and the explicit connection/reconnect services.
6. Verify simulator and device behavior for Metal rendering, reconnect, tmux multi-pane ownership, and daemon-backed attach flows.

## Phase 3 Gate

Do not declare Phase 2 complete until all of the following are true:

1. The iOS app builds cleanly from this repository on macOS with Xcode.
2. The app can browse daemon-backed sessions and attach to an existing tmux session through the current daemon API.
3. Background → foreground reconnect is verified on device without regressing the current upstream ordering guarantees.
4. tmux session-name resolution keeps the UUID-sentinel behavior and its regression coverage.
5. Surface ownership changes are stable under pane closes, splits, and primary-surface promotion.

Linux can prepare the scaffold and tighten the plan, but Linux alone cannot honestly sign off Phase 2 as complete.
