#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ghostty_root="$repo_root/_upstream/ghostty"
framework_root="$repo_root/ios/iTTY/Frameworks"
xcframework_path="$ghostty_root/macos/GhosttyKit.xcframework"
mounted_metal_bin="/Volumes/MetalToolchainCryptex/Metal.xctoolchain/usr/bin/metal"
zig_bin=""

resolve_zig() {
  if [[ -n "${ZIG_BIN:-}" ]]; then
    zig_bin="$ZIG_BIN"
    return
  fi

  if [[ -x /opt/homebrew/bin/zig ]]; then
    zig_bin="/opt/homebrew/bin/zig"
    return
  fi

  zig_bin="$(command -v zig)"
}

require_zig() {
  local zig_version
  resolve_zig
  zig_version="$("$zig_bin" version)"
  if [[ "$zig_version" != "0.15.2" ]]; then
    echo "Expected zig 0.15.2, found $zig_version" >&2
    exit 1
  fi
}

apply_patch_if_needed() {
  local patch_path="$1"

  if git -C "$ghostty_root" apply --check "$patch_path" >/dev/null 2>&1; then
    git -C "$ghostty_root" apply "$patch_path"
    return
  fi

  if git -C "$ghostty_root" apply --reverse --check "$patch_path" >/dev/null 2>&1; then
    return
  fi

  echo "Ghostty patch does not apply cleanly: $patch_path" >&2
  exit 1
}

apply_patches() {
  local patch_path
  shopt -s nullglob
  for patch_path in "$repo_root"/patches/ghostty/*.patch; do
    apply_patch_if_needed "$patch_path"
  done
}

ensure_metal_toolchain() {
  if xcrun -sdk iphoneos metal -v >/dev/null 2>&1; then
    return
  fi

  if [[ ! -x "$mounted_metal_bin" ]]; then
    xcodebuild -downloadComponent MetalToolchain >/dev/null
  fi

  if [[ ! -x "$mounted_metal_bin" ]]; then
    local dmg_path
    dmg_path="$(find /System/Library/AssetsV2/com_apple_MobileAsset_MetalToolchain -path '*/AssetData/Restore/*.dmg' -print | head -n 1)"
    if [[ -z "$dmg_path" ]]; then
      echo "Unable to locate the downloaded Metal Toolchain DMG" >&2
      exit 1
    fi

    if [[ ! -d /Volumes/MetalToolchainCryptex ]]; then
      hdiutil attach -nobrowse -readonly "$dmg_path" >/dev/null
    fi
  fi

  if [[ ! -x "$mounted_metal_bin" ]]; then
    echo "Metal Toolchain is still unavailable after download/mount" >&2
    exit 1
  fi
}

install_xcframework() {
  mkdir -p "$framework_root"
  rm -rf "$framework_root/GhosttyKit.xcframework"
  cp -R "$xcframework_path" "$framework_root/"
  find "$framework_root/GhosttyKit.xcframework" -path '*/Headers/module.modulemap' -print0 |
    while IFS= read -r -d '' modulemap; do
      mv "$modulemap" "${modulemap%module.modulemap}GhosttyKit.modulemap"
    done
}

require_zig
ensure_metal_toolchain
apply_patches

(
  cd "$ghostty_root"
  "$zig_bin" build -Demit-xcframework=true -Dxcframework-target=universal -Demit-macos-app=false
)

install_xcframework

echo "GhosttyKit installed to $framework_root/GhosttyKit.xcframework"
