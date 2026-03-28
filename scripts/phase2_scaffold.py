#!/usr/bin/env python3
"""Validate and scaffold the Linux-safe Phase 2 iOS handoff."""

from __future__ import annotations

import argparse
import json
import shutil
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
UPSTREAM_APP_ROOT = REPO_ROOT / "_upstream" / "geistty" / "Geistty"
UPSTREAM_SOURCES_ROOT = UPSTREAM_APP_ROOT / "Sources"
UPSTREAM_UNIT_TESTS_ROOT = UPSTREAM_APP_ROOT / "GeisttyTests"
UPSTREAM_UI_TESTS_ROOT = UPSTREAM_APP_ROOT / "GeisttyUITests"

DEST_APP_ROOT = REPO_ROOT / "ios" / "iTTY"
DEST_SOURCES_ROOT = DEST_APP_ROOT / "Sources"
DEST_UNIT_TESTS_ROOT = REPO_ROOT / "ios" / "iTTYTests" / "UpstreamParity"
DEST_UI_TESTS_ROOT = REPO_ROOT / "ios" / "iTTYUITests" / "UpstreamParity"
DEST_ASSETS_ROOT = DEST_APP_ROOT / "Assets.xcassets"
DEST_RESOURCES_ROOT = DEST_APP_ROOT / "Resources"
MANIFEST_PATH = REPO_ROOT / "ios" / "phase2-manifest.json"

SOURCE_MAP = [
    {
        "source": "App/GeisttyApp.swift",
        "destination": "App/iTTYApp.swift",
        "action": "Rename + extend",
    },
    {
        "source": "App/ContentView.swift",
        "destination": "App/ContentView.swift",
        "action": "Replace (new nav)",
    },
    {
        "source": "Auth/ConnectionProfile.swift",
        "destination": "Core/Auth/ConnectionProfile.swift",
        "action": "Keep",
    },
    {
        "source": "Auth/CredentialProvider.swift",
        "destination": "Core/Auth/CredentialProvider.swift",
        "action": "Keep",
    },
    {
        "source": "Auth/KeychainManager.swift",
        "destination": "Core/Auth/KeychainManager.swift",
        "action": "Keep",
    },
    {
        "source": "Auth/SSHKeyManager.swift",
        "destination": "Core/Auth/SSHKeyManager.swift",
        "action": "Keep",
    },
    {
        "source": "Auth/SSHKeyParser.swift",
        "destination": "Core/Auth/SSHKeyParser.swift",
        "action": "Keep",
    },
    {
        "source": "Auth/BiometricGatekeeper.swift",
        "destination": "Core/Auth/BiometricGatekeeper.swift",
        "action": "Keep",
    },
    {
        "source": "Ghostty/Ghostty.swift",
        "destination": "Core/Terminal/SurfaceView.swift",
        "action": "Refactor (extract)",
    },
    {
        "source": "Ghostty/Ghostty.App.swift",
        "destination": "Core/Terminal/GhosttyRuntime.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/Ghostty.Config.swift",
        "destination": "Core/Terminal/GhosttyConfig.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/Ghostty.Command.swift",
        "destination": "Core/Terminal/Commands.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/Ghostty.SearchState.swift",
        "destination": "Core/Terminal/SearchState.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/GhosttyInput.swift",
        "destination": "Core/Terminal/InputTranslation.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/FontMapping.swift",
        "destination": "Core/Config/FontMapping.swift",
        "action": "Move",
    },
    {
        "source": "Ghostty/ConfigSyncManager.swift",
        "destination": "Core/Config/ConfigSyncManager.swift",
        "action": "Move",
    },
    {
        "source": "Ghostty/SurfaceSearchOverlay.swift",
        "destination": "Core/Terminal/SearchOverlay.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/SelectionOverlay.swift",
        "destination": "Core/Terminal/SelectionOverlay.swift",
        "action": "Rename",
    },
    {
        "source": "Ghostty/TmuxSurfaceProtocol.swift",
        "destination": "Core/Tmux/TmuxSurfaceProtocol.swift",
        "action": "Move",
    },
    {
        "source": "Ghostty/Ghostty.SurfaceConfiguration.swift",
        "destination": "Core/Terminal/SurfaceConfiguration.swift",
        "action": "Rename",
    },
    {
        "source": "SSH/SSHSession.swift",
        "destination": "Core/SSH/SSHSession.swift",
        "action": "Refactor",
    },
    {
        "source": "SSH/NIOSSHConnection.swift",
        "destination": "Core/SSH/NIOSSHConnection.swift",
        "action": "Keep",
    },
    {
        "source": "SSH/SSHCommandRunner.swift",
        "destination": "Core/SSH/SSHCommandRunner.swift",
        "action": "Keep",
    },
    {
        "source": "SSH/TmuxSessionManager.swift",
        "destination": "Core/Tmux/TmuxSessionManager.swift",
        "action": "Refactor (extract registry)",
    },
    {
        "source": "SSH/TmuxLayout.swift",
        "destination": "Core/Tmux/TmuxLayout.swift",
        "action": "Keep",
    },
    {
        "source": "SSH/TmuxSplitTree.swift",
        "destination": "Core/Tmux/TmuxSplitTree.swift",
        "action": "Keep",
    },
    {
        "source": "SSH/TmuxModels.swift",
        "destination": "Core/Tmux/TmuxModels.swift",
        "action": "Keep",
    },
    {
        "source": "SSH/TmuxSessionNameResolver.swift",
        "destination": "Core/Tmux/TmuxSessionNameResolver.swift",
        "action": "Keep (preserve UUID sentinel)",
    },
    {
        "source": "SSH/TmuxWireDiagnostics.swift",
        "destination": "Core/Tmux/TmuxWireDiagnostics.swift",
        "action": "Keep",
    },
    {
        "source": "Terminal/TerminalContainerView.swift",
        "destination": "Features/Terminal/TerminalContainerView.swift",
        "action": "Refactor",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+Keyboard.swift",
        "destination": "Features/Terminal/Keyboard.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+MenuBar.swift",
        "destination": "Features/Terminal/MenuBar.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+Search.swift",
        "destination": "Features/Terminal/Search.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+Shortcuts.swift",
        "destination": "Features/Terminal/Shortcuts.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+Tmux.swift",
        "destination": "Features/Terminal/TmuxPane.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+StatusBar.swift",
        "destination": "Features/Terminal/StatusBar.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/RawTerminalUIViewController+WindowPicker.swift",
        "destination": "Features/Terminal/WindowPicker.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/Theme.swift",
        "destination": "Core/Config/ThemeManager.swift",
        "action": "Move",
    },
    {
        "source": "Terminal/TmuxMultiPaneView.swift",
        "destination": "Features/Terminal/MultiPaneView.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/TmuxSplitView.swift",
        "destination": "Features/Terminal/SplitView.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/TmuxWindowPickerView.swift",
        "destination": "Features/Terminal/WindowPickerView.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/TmuxSessionPickerView.swift",
        "destination": "Features/Terminal/SessionPickerView.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/TmuxStatusBarView.swift",
        "destination": "Features/Terminal/StatusBarView.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/CommandPaletteView.swift",
        "destination": "Features/Terminal/CommandPalette.swift",
        "action": "Rename",
    },
    {
        "source": "Terminal/TerminalToolbar.swift",
        "destination": "Features/Terminal/Toolbar.swift",
        "action": "Rename",
    },
    {
        "source": "UI/ConnectionListView.swift",
        "destination": "Features/Machines/ConnectionListView.swift",
        "action": "Move (evolves into MachineListView)",
    },
    {
        "source": "UI/ConnectionEditorView.swift",
        "destination": "Features/Machines/ConnectionEditorView.swift",
        "action": "Move (evolves into AddMachineView)",
    },
    {
        "source": "UI/SettingsView.swift",
        "destination": "Features/Settings/SettingsView.swift",
        "action": "Move",
    },
    {
        "source": "UI/KeyTableIndicatorView.swift",
        "destination": "Features/Terminal/KeyTableIndicator.swift",
        "action": "Move",
    },
]

PLANNED_NEW_FILES = [
    "Features/Machines/TailscaleDiscoveryView.swift",
    "Features/Terminal/TerminalTabBar.swift",
    "Features/Onboarding/OnboardingView.swift",
    "Services/Tailscale/TailscaleAPIClient.swift",
    "Services/Tailscale/TailscaleVPNDetector.swift",
    "Services/Connection/ConnectionManager.swift",
    "Services/Connection/AutoReconnectService.swift",
]


class ValidationError(RuntimeError):
    """Raised when the scaffold manifest and upstream sources disagree."""


def filecmp_bytes(left: Path, right: Path) -> bool:
    return left.read_bytes() == right.read_bytes()


def list_upstream_source_files() -> list[str]:
    return sorted(str(path.relative_to(UPSTREAM_SOURCES_ROOT)) for path in UPSTREAM_SOURCES_ROOT.rglob("*.swift"))


def list_upstream_test_files(root: Path) -> list[str]:
    return sorted(str(path.relative_to(root)) for path in root.glob("*.swift"))


def validate_source_map() -> dict[str, int]:
    errors: list[str] = []
    actual_sources = set(list_upstream_source_files())
    mapped_sources = [entry["source"] for entry in SOURCE_MAP]
    mapped_destinations = [entry["destination"] for entry in SOURCE_MAP]

    missing_from_upstream = sorted(set(mapped_sources) - actual_sources)
    extra_in_upstream = sorted(actual_sources - set(mapped_sources))

    if missing_from_upstream:
        errors.append(f"mapped sources missing upstream: {missing_from_upstream}")
    if extra_in_upstream:
        errors.append(f"upstream sources missing from map: {extra_in_upstream}")
    if len(mapped_sources) != len(set(mapped_sources)):
        errors.append("duplicate source entries detected in SOURCE_MAP")
    if len(mapped_destinations) != len(set(mapped_destinations)):
        errors.append("duplicate destination entries detected in SOURCE_MAP")

    for entry in SOURCE_MAP:
        source_path = UPSTREAM_SOURCES_ROOT / entry["source"]
        if not source_path.exists():
            errors.append(f"missing upstream source: {entry['source']}")

    if errors:
        raise ValidationError("\n".join(errors))

    return {
        "source_files": len(mapped_sources),
        "unit_test_files": len(list_upstream_test_files(UPSTREAM_UNIT_TESTS_ROOT)),
        "ui_test_files": len(list_upstream_test_files(UPSTREAM_UI_TESTS_ROOT)),
        "planned_new_files": len(PLANNED_NEW_FILES),
    }


def write_manifest() -> None:
    manifest = {
        "manifest_version": 1,
        "source": "github.com/daiimus/geistty v0.1-stable",
        "upstream_project_root": "_upstream/geistty/Geistty",
        "upstream_xcode_project": "_upstream/geistty/Geistty/Geistty.xcodeproj",
        "ios_root": "ios",
        "destination_root": "ios/iTTY",
        "source_destination_root": "ios/iTTY/Sources",
        "unit_tests_destination_root": "ios/iTTYTests/UpstreamParity",
        "ui_tests_destination_root": "ios/iTTYUITests/UpstreamParity",
        "assets_destination_root": "ios/iTTY/Assets.xcassets",
        "resources_destination_root": "ios/iTTY/Resources",
        "planned_new_files": PLANNED_NEW_FILES,
        "source_map": SOURCE_MAP,
    }
    MANIFEST_PATH.parent.mkdir(parents=True, exist_ok=True)
    MANIFEST_PATH.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")


def copy_file(source: Path, destination: Path, overwrite: bool) -> str:
    destination.parent.mkdir(parents=True, exist_ok=True)

    if destination.exists():
        if filecmp_bytes(source, destination):
            return "unchanged"
        if not overwrite:
            return "preserved"
        shutil.copy2(source, destination)
        return "updated"

    shutil.copy2(source, destination)
    return "copied"


def copy_tree(source_root: Path, destination_root: Path, overwrite: bool) -> dict[str, int]:
    counts = {"copied": 0, "updated": 0, "unchanged": 0, "preserved": 0}
    for source in sorted(path for path in source_root.rglob("*") if path.is_file()):
        destination = destination_root / source.relative_to(source_root)
        outcome = copy_file(source, destination, overwrite)
        counts[outcome] += 1
    return counts


def scaffold(overwrite: bool) -> None:
    validate_source_map()
    write_manifest()

    source_counts = {"copied": 0, "updated": 0, "unchanged": 0, "preserved": 0}
    for entry in SOURCE_MAP:
        source = UPSTREAM_SOURCES_ROOT / entry["source"]
        destination = DEST_SOURCES_ROOT / entry["destination"]
        outcome = copy_file(source, destination, overwrite)
        source_counts[outcome] += 1

    unit_counts = copy_tree(UPSTREAM_UNIT_TESTS_ROOT, DEST_UNIT_TESTS_ROOT, overwrite)
    ui_counts = copy_tree(UPSTREAM_UI_TESTS_ROOT, DEST_UI_TESTS_ROOT, overwrite)
    asset_counts = copy_tree(UPSTREAM_APP_ROOT / "Assets.xcassets", DEST_ASSETS_ROOT, overwrite)
    resource_counts = copy_tree(UPSTREAM_APP_ROOT / "Resources", DEST_RESOURCES_ROOT, overwrite)

    print("Phase 2 scaffold refreshed:")
    print(f"  sources: {source_counts}")
    print(f"  unit tests: {unit_counts}")
    print(f"  ui tests: {ui_counts}")
    print(f"  assets: {asset_counts}")
    print(f"  resources: {resource_counts}")
    print(f"  manifest: {MANIFEST_PATH.relative_to(REPO_ROOT)}")


def verify_scaffold() -> None:
    summary = validate_source_map()
    missing: list[str] = []

    for entry in SOURCE_MAP:
        destination = DEST_SOURCES_ROOT / entry["destination"]
        if not destination.exists():
            missing.append(str(destination.relative_to(REPO_ROOT)))

    for root, destination_root in [
        (UPSTREAM_UNIT_TESTS_ROOT, DEST_UNIT_TESTS_ROOT),
        (UPSTREAM_UI_TESTS_ROOT, DEST_UI_TESTS_ROOT),
        (UPSTREAM_APP_ROOT / "Assets.xcassets", DEST_ASSETS_ROOT),
        (UPSTREAM_APP_ROOT / "Resources", DEST_RESOURCES_ROOT),
    ]:
        for source in sorted(path for path in root.rglob("*") if path.is_file()):
            destination = destination_root / source.relative_to(root)
            if not destination.exists():
                missing.append(str(destination.relative_to(REPO_ROOT)))

    if not MANIFEST_PATH.exists():
        missing.append(str(MANIFEST_PATH.relative_to(REPO_ROOT)))

    if missing:
        raise ValidationError("scaffold is incomplete:\n" + "\n".join(missing[:50]))

    print("Phase 2 scaffold verified:")
    print(f"  sources mapped: {summary['source_files']}")
    print(f"  unit tests copied: {summary['unit_test_files']}")
    print(f"  ui tests copied: {summary['ui_test_files']}")
    print(f"  planned iTTY-only files remaining: {summary['planned_new_files']}")


def check_mapping() -> None:
    summary = validate_source_map()
    print("Phase 2 mapping is internally consistent:")
    print(f"  upstream sources mapped: {summary['source_files']}")
    print(f"  upstream unit tests available: {summary['unit_test_files']}")
    print(f"  upstream ui tests available: {summary['ui_test_files']}")
    print(f"  planned iTTY-only files: {summary['planned_new_files']}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("check", help="validate the upstream source map")
    subparsers.add_parser("verify", help="validate the copied ios scaffold exists")
    subparsers.add_parser("export-manifest", help="rewrite ios/phase2-manifest.json")

    scaffold_parser = subparsers.add_parser("scaffold", help="copy upstream files into ios/")
    scaffold_parser.add_argument(
        "--overwrite",
        action="store_true",
        help="overwrite files in ios/ when they differ from upstream",
    )

    return parser.parse_args()


def main() -> int:
    args = parse_args()

    try:
        if args.command == "check":
            check_mapping()
        elif args.command == "verify":
            verify_scaffold()
        elif args.command == "export-manifest":
            validate_source_map()
            write_manifest()
            print(f"wrote {MANIFEST_PATH.relative_to(REPO_ROOT)}")
        elif args.command == "scaffold":
            scaffold(overwrite=args.overwrite)
        else:
            raise ValidationError(f"unknown command: {args.command}")
    except ValidationError as exc:
        print(str(exc), file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
