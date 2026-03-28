# Daemon Configuration

## File Location

The daemon reads and writes config at:

```text
~/.config/itty/config.toml
```

`XDG_CONFIG_HOME` is respected when set.

## Defaults

```toml
listen_addr = ":8080"
tmux_path = "tmux"
auto_wrap = true
tailscale_serve = true
apns_key_path = ""
apns_key_id = ""
apns_team_id = ""
```

## Behavior

- missing config file: daemon uses defaults
- invalid config file: daemon startup fails with an error
- config updates: written atomically through a temp file and rename
- validation: `listen_addr` and `tmux_path` must be non-empty

## Phase 1 Notes

- `PUT /config/auto` updates both the shell rc file and `config.toml`
- APNs fields are persisted but not used yet
- `tailscale_serve` is persisted but not acted on yet
