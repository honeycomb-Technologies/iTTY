# Machines

## Current Phase 2 Status

The Linux-side Phase 2 pass now includes a first machine-management layer in the iOS scaffold:

- `ios/iTTY/Sources/Models/Machine.swift`
- `ios/iTTY/Sources/Features/Machines/MachineListView.swift`
- `ios/iTTY/Sources/Features/Machines/AddMachineView.swift`

## Scope

Machines represent daemon endpoints, not active SSH sessions. Each machine stores:

- daemon scheme, host, and port
- optional link to a saved SSH connection profile
- favorite state and last-seen metadata

## Current Flow

1. User adds a machine with daemon host/port.
2. The machine list opens a daemon-backed session browser.
3. The linked SSH profile is available for the later attach flow.

The actual attach handoff from daemon-selected session to SSH/tmux connection still requires macOS integration work.
