# Engineering Standards

This document sets the minimum bar for all post-sweep work on iTTY. It is intentionally strict. If a change does not meet these standards, it is not done.

## Non-Negotiables

- Code, docs, and shipped behavior must agree in the same change.
- Public API work requires explicit tests for success and failure cases.
- Placeholder behavior does not ship as if it were real functionality.
- A feature is not considered implemented until it is verified end to end.
- Small, finished vertical slices are preferred over broad speculative scaffolding.

## Repo Truth

- The source of truth is the actual repo, not the aspirational architecture doc.
- If the repo shape changes, `README.md`, `CLAUDE.md`, and any affected docs must be updated in the same patch.
- Do not document endpoints, commands, directories, or workflows that do not exist yet without clearly marking them as planned.
- `_upstream/` is reference material only. It must not be confused with implemented product code.

## API Standards

- Every externally consumed response must have explicit `json` tags.
- Public JSON schemas must be stable and intentional. No leaking Go field naming by accident.
- Handlers must validate inputs and map errors to deliberate status codes.
- No `501` or stub public endpoints on `main` unless the route is clearly hidden from user-facing docs.
- API docs must be updated in the same patch as any route or schema change.

## Process and Failure Handling

- Startup must fail fast on fatal dependency or bind errors.
- Shutdown paths must be bounded, observable, and testable.
- Silent error swallowing is not acceptable in core paths.
- If a parser drops malformed data, the decision must be explicit, documented, and tested.
- Every external process call must have a timeout and context.

## tmux and Shell Safety

- Never interpolate untrusted strings into tmux format expressions or shell snippets without escaping.
- Shell config mutation must remain idempotent, reversible, and narrowly scoped to marked blocks.
- Session names, rc file paths, and command output parsing must be treated as hostile inputs.
- Platform-specific fallback behavior must distinguish unsupported behavior from actual execution failure.

## Testing Bar

- New exported behavior requires unit tests.
- Parsing logic requires table-driven tests with malformed input cases.
- HTTP handlers require tests for both happy paths and status/error contracts.
- Config and shell mutation code require temp-dir based tests, never tests against a real home directory.
- A passing `go test` run with zero tests does not count as validation.

## Complexity Rules

- Do not introduce abstractions until two concrete call sites justify them.
- Prefer standard library solutions unless there is a demonstrated gap.
- Avoid hidden global state.
- Keep package responsibilities narrow and legible.
- Comments should explain non-obvious intent, not narrate obvious code.

## Documentation Rules

- Replace placeholders as soon as the underlying code exists.
- Do not leave stale docs behind after a refactor.
- Architecture docs may describe target direction, but implementation docs must describe what exists now.
- Every phase doc must state what is complete, what is not, and what blocks sign-off.

## Definition Of Done

A change is done only when:

1. The implementation is correct and minimal.
2. Tests prove the behavior and the important failure cases.
3. Docs reflect the actual new state.
4. Build, test, and lint pass.
5. The diff removes ambiguity instead of adding it.
