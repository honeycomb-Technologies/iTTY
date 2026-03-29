# Building

## Prerequisites

- macOS with Xcode 16+ (Swift 5.9+)
- [xcodegen](https://github.com/yonaskolb/XcodeGen) (`brew install xcodegen`)
- Zig 0.15.2 (for building GhosttyKit)
  - `scripts/build_ghosttykit.sh` prefers Homebrew Zig at `/opt/homebrew/bin/zig` when available.
  - Override with `ZIG_BIN=/absolute/path/to/zig` if needed.

## Xcode Project

The Xcode project is generated from `ios/project.yml` using xcodegen. Do not edit `iTTY.xcodeproj` directly.

```bash
cd ios
xcodegen generate
```

This creates three targets:
- **iTTY** — iOS app (iPhone + iPad, iOS 17+)
- **iTTYTests** — Unit tests
- **iTTYUITests** — UI tests

Current verified build state:
- Apple silicon simulator build is verified with `CODE_SIGNING_ALLOWED=NO`
- GhosttyKit currently ships an `ios-arm64-simulator` slice, so `x86_64` simulator builds are intentionally excluded in `ios/project.yml`
- Signed device builds still need local validation on hardware

### SPM Dependencies

Resolved automatically on first build:
- [swift-nio-ssh](https://github.com/daiimus/swift-nio-ssh) (branch `add-rsa-support`) — SSH protocol with RSA support
- [swift-nio-transport-services](https://github.com/apple/swift-nio-transport-services) (>=1.20.0) — Network.framework transport

## GhosttyKit Framework

GhosttyKit is a pre-built XCFramework containing Ghostty's Zig terminal engine. It is not checked into the repo and must be built from the Ghostty fork.

Validated source baseline:
- Fork: `github.com/daiimus/ghostty`
- Branch: `ios-external-backend`
- Commit: `21c717340b62349d67124446c2447bf38796540b`
- Repo-owned patches: `patches/ghostty/*.patch`

### Building GhosttyKit

```bash
# Clone the Ghostty fork (if not already present)
git clone --branch ios-external-backend https://github.com/daiimus/ghostty.git _upstream/ghostty

# Apply the checked-in Ghostty patches, build the XCFramework, copy it into
# ios/iTTY/Frameworks, and rename module maps to GhosttyKit.modulemap
./scripts/build_ghosttykit.sh
```

The build script is the source of truth. It:

- applies every patch in `patches/ghostty/` to `_upstream/ghostty`
- downloads or mounts the Metal Toolchain if Xcode's `metal` shim is unavailable
- builds GhosttyKit with `-Demit-xcframework=true -Dxcframework-target=universal -Demit-macos-app=false`
- copies `macos/GhosttyKit.xcframework` into `ios/iTTY/Frameworks`
- renames each `module.modulemap` to `GhosttyKit.modulemap`

If you need to run the raw Ghostty build yourself, do it from `_upstream/ghostty`:

```bash
/opt/homebrew/bin/zig build \
  -Demit-xcframework=true \
  -Dxcframework-target=universal \
  -Demit-macos-app=false
```

After refreshing GhosttyKit, regenerate the Xcode project:

```bash
cd ios && xcodegen generate
```

## Building the App

```bash
# Simulator build
xcodebuild -project ios/iTTY.xcodeproj -scheme iTTY \
  -destination 'generic/platform=iOS Simulator' \
  CODE_SIGNING_ALLOWED=NO \
  build

# Device build (requires signing; command is documented but still needs local validation)
xcodebuild -project ios/iTTY.xcodeproj -scheme iTTY \
  -destination 'platform=iOS,name=YourDevice' \
  -allowProvisioningUpdates build
```

## Running Tests

```bash
# Unit tests
xcodebuild -project ios/iTTY.xcodeproj -scheme iTTY \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  test

# UI tests require a connected SSH server (see TestConfig.example.swift)
```

## Project Structure

See `docs/ios/project-structure.md` for the full directory layout and file organization.
