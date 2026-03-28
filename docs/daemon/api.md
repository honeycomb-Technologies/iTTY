# iTTY Daemon API Reference

Base URL: `http://localhost:8080`

## Implemented Endpoints

### GET /health

Returns daemon status and tmux availability.

**Response**
```json
{
  "status": "ok",
  "version": "0.1.0",
  "platform": "linux/amd64",
  "tmuxInstalled": true,
  "tmuxVersion": "3.6a"
}
```

### GET /sessions

Lists all tmux sessions with summary metadata.

### GET /sessions/{name}

Returns a single tmux session with its windows and panes.

Returns `404` when the session does not exist.

### GET /sessions/{name}/content

Captures the visible content of the session's active pane.

**Response**
```json
{
  "content": "~ ❯ ls\nREADME.md  src/  tests/\n~ ❯ "
}
```

### GET /config

Returns the daemon configuration currently loaded in memory.

**Response**
```json
{
  "listenAddr": ":8080",
  "tmuxPath": "tmux",
  "autoWrap": true,
  "tailscaleServe": true,
  "apnsKeyPath": "",
  "apnsKeyID": "",
  "apnsTeamID": ""
}
```

### PUT /config/auto

Enables or disables tmux shell auto-wrap and persists the updated config.

**Request**
```json
{
  "enabled": true
}
```

**Response**
```json
{
  "listenAddr": ":8080",
  "tmuxPath": "tmux",
  "autoWrap": true,
  "tailscaleServe": true,
  "apnsKeyPath": "",
  "apnsKeyID": "",
  "apnsTeamID": ""
}
```

Possible errors:

- `400` for invalid JSON or missing `enabled`
- `500` for shell detection, shell mutation, or config save failures

### GET /windows

Lists open terminal windows detected on the local desktop.

**Response**
```json
[
  {
    "id": "0x04800007",
    "title": "ghostty — ~/project",
    "app": "ghostty",
    "focused": true
  }
]
```

## Not In Phase 1

The following are not implemented in the Phase 1 daemon:

- WebSocket streaming
- APNs device registration
- Tailscale control-plane integration
