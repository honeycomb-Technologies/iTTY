# Sessions

## Current Phase 2 Status

The iOS scaffold now includes the first daemon-backed session browser layer:

- `ios/iTTY/Sources/Features/SessionBrowser/SessionBrowserView.swift`
- `ios/iTTY/Sources/Features/SessionBrowser/SessionRowView.swift`
- `ios/iTTY/Sources/Services/Daemon/DaemonClient.swift`
- `ios/iTTY/Sources/Models/SavedSession.swift`

## Behavior

The browser targets the implemented Phase 1 daemon API:

- `GET /health`
- `GET /sessions`
- `GET /sessions/{name}`
- `GET /sessions/{name}/content`

It can:

- fetch daemon health
- list tmux-backed sessions
- inspect per-session windows and panes
- preview the active pane content
- hand a selected daemon session off to the existing SSH/tmux flow when the machine has a linked connection profile

## Remaining Work

- wire the browser into a real Xcode target
- validate the linked-profile attach flow under Xcode and fix any compile/runtime issues
- validate the browse flow on simulator and device
