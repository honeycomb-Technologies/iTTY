# Daemon Architecture

## Scope

The Phase 1 daemon is a single Go binary that:

- reads config from `~/.config/itty/config.toml`
- inspects tmux state
- exposes that state over HTTP
- can toggle shell auto-wrap on and off
- reports desktop terminal windows through platform-specific helpers

## Packages

- `cmd/itty`: CLI entrypoint and process lifecycle
- `internal/api`: HTTP routes and dependency wiring
- `internal/config`: default config, validation, load, and save
- `internal/shell`: shell detection plus rc-file mutation
- `internal/tmux`: tmux command execution and output parsing
- `internal/platform`: Linux and macOS terminal window discovery

## Request Flow

### Sessions

1. HTTP handler calls the tmux client
2. tmux client executes the required tmux command with a timeout
3. tmux output is parsed into strict Go structs
4. API returns explicit JSON

### Auto-Wrap Toggle

1. `PUT /config/auto` validates `{"enabled": <bool>}`
2. shell manager detects the user shell and rc file
3. daemon configures or unconfigures the marked rc block
4. config is updated and written back to disk
5. API returns the persisted config shape

## Lifecycle

- `itty start` loads config first
- bind errors fail startup immediately
- signal handling triggers bounded shutdown with a 5 second timeout
- HTTP shutdown uses `http.ErrServerClosed` as the normal exit path

## Design Constraints

- tmux is the persistence layer, not a custom session service
- shell mutation must be reversible and idempotent
- config must remain valid and writable even if the file does not exist yet
- public JSON contracts must not depend on Go default field naming
