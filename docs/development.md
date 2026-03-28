# Development

## Current Repo State

Phase 1 is the desktop daemon. The repository currently contains:

- `daemon/` for the Go daemon
- `docs/` for the implementation docs and roadmap
- `ios/` for the Linux-generated iOS scaffold and parity assets/tests
- `_upstream/geistty/` as reference material and the source for scaffold verification

There is still no buildable `ios/` app target in this repo yet.

## Working Rules

- Read `docs/roadmap.md` before broad changes
- Read `docs/engineering-standards.md` before Phase 1 work
- Keep docs and code aligned in the same patch
- Do not leave public stubs or undocumented behavior on `main`

## Daemon Commands

```bash
cd daemon
make build
make test
make lint
make run
make cross
make clean
```

Prefer `make build` over bare `go build ./cmd/itty` so output stays in `daemon/bin/`.

## iOS Scaffold Commands

```bash
make scaffold-ios
make check-ios-scaffold
```

These commands prepare and verify the iOS scaffold on Linux. Actual app builds remain macOS-only.

## Verification

Minimum verification for daemon changes:

```bash
cd daemon
make test
make lint
```

Good manual smoke checks:

```bash
cd daemon
./bin/itty status
curl -s localhost:8080/health | python3 -m json.tool
curl -s localhost:8080/sessions | python3 -m json.tool
curl -s localhost:8080/config | python3 -m json.tool
curl -s localhost:8080/windows | python3 -m json.tool
```

## Test Expectations

- Add unit tests for every new exported behavior
- Add table-driven tests for parsing logic
- Add handler tests for status codes and JSON contracts
- Use temp directories for shell and config tests
- Treat zero-test passes as insufficient validation
