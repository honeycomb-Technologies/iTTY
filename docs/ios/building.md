# Building

## Current State

As of 2026-03-28, this repository contains a verified iOS scaffold under `ios/`, but it does not yet contain a buildable Xcode target.

Linux-safe preparation:

```bash
make scaffold-ios
make check-ios-scaffold
```

These commands copy and verify:

- `ios/iTTY/Sources`
- `ios/iTTY/Assets.xcassets`
- `ios/iTTY/Resources`
- `ios/iTTYTests/UpstreamParity`
- `ios/iTTYUITests/UpstreamParity`

`make scaffold-ios` is intentionally conservative: it preserves files that have diverged from the upstream scaffold unless the script is run manually with `--overwrite`.

## Mac-Only Build Work

The following still has to happen on macOS with Xcode:

1. Create or rename the real iTTY app, test, and UI test targets.
2. Wire the files in `ios/` into the Xcode project.
3. Fix symbol and module naming still inherited from upstream Geistty.
4. Verify Metal rendering, reconnect behavior, and tmux flows on simulator and device.

Use `_upstream/geistty/Geistty/Geistty.xcodeproj` as the project-configuration reference while creating the real iTTY project.
