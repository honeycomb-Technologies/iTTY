# Daemon Integration

## Current Contract

Phase 1 established the daemon-side HTTP contract. Phase 2 iOS work should target the existing API in `docs/daemon/api.md` without inventing a parallel transport.

Current endpoints:

- `GET /health`
- `GET /sessions`
- `GET /sessions/{name}`
- `GET /sessions/{name}/content`
- `GET /config`
- `PUT /config/auto`
- `GET /windows`

There is no WebSocket stream yet. Real-time updates remain a later-phase concern.

## Phase 2 Responsibilities

The initial iOS daemon client should:

1. Discover and validate the daemon with `GET /health`.
2. Fetch the session list with `GET /sessions`.
3. Present a session browser using the daemon response, not tmux shell parsing on-device.
4. Fetch per-session metadata or pane content on demand with the session detail endpoints.
5. Keep request/response models explicit and versioned inside the iOS app.

The current Phase 2 scaffold now includes that client/model/browser layer in source form. What remains is Xcode wiring and runtime validation.

## Boundaries

- The daemon reports tmux-backed desktop state. It does not replace the existing SSH/tmux transport used by the Geistty foundation.
- iOS should treat daemon failures as recoverable browse-state failures, not as a reason to tear down an active terminal connection.
- If the daemon is unreachable, the app should degrade to direct connection flows instead of blocking the entire terminal product.

## Phase 3 Follow-On

These are not required for Phase 2 completion:

- WebSocket or streaming session updates
- APNs-triggered reconnect or wake flows
- Tailscale control-plane automation
- notification-driven daemon discovery
