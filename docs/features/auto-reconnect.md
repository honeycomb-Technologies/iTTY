# Auto Reconnect

## Verified Upstream Baseline

Geistty already contains reconnect behavior across multiple layers:

- `ContentView` uses `scenePhase` gating.
- `TerminalContainerView` handles app-active/background notifications.
- `SSHSession` owns the actual reconnect and tmux reattach work.

The problem is not a missing reconnect feature. The problem is that the ownership is spread across app and transport layers, which makes refactoring easy to break.

## Phase 2 Goal

Phase 2 should extract a clearer owner for reconnect behavior, most likely `ConnectionManager` or `AutoReconnectService`, while preserving upstream semantics:

1. Backgrounding can suspend or kill the SSH transport.
2. Foregrounding should restore the SSH → tmux pipeline.
3. Existing ordering guarantees must survive the refactor.
4. UUID-based tmux session resolution must not regress during reconnect.

## Acceptance Criteria

Do not call reconnect complete until all of the following are verified on device:

1. Background → foreground returns to the prior tmux session.
2. Reconnect does not attach to the wrong session.
3. tmux multi-pane state survives reconnect.
4. Active write failures reach visible UI state instead of silently stalling.

## Non-Goals for Phase 2

- Inventing a brand-new reconnect algorithm
- Adding push-triggered wake/reconnect flows
- Treating transient SSH retries as a substitute for lifecycle-aware reconnect ownership
