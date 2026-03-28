# Daemon Install

## Current State

Phase 1 does not ship a one-command installer yet. Installation is currently manual from source.

## Build From Source

```bash
git clone <repo>
cd iTTY/daemon
make build
```

The daemon binary is written to:

```text
daemon/bin/itty
```

## Run

```bash
cd daemon
./bin/itty start
```

## Optional Local Install

```bash
cd daemon
make install
```

This installs the binary to your Go bin directory.

## Phase 1 Limitations

- no launchd or systemd packaging in repo yet
- no installer script in repo yet
- no automatic Tailscale Serve configuration yet
