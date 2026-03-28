# iTTY Daemon API Reference

Base URL: `http://localhost:8080` (or `https://<machine>.tailnet:8080` via Tailscale Serve)

## Endpoints

### GET /health

Returns daemon status and environment info.

**Response:**
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

Lists all tmux sessions with metadata.

**Response:**
```json
[
  {
    "name": "itty-pts-1",
    "windows": 2,
    "created": "2026-03-28T10:30:00-07:00",
    "attached": true,
    "lastPaneCommand": "nvim",
    "lastPanePath": "/home/user/project"
  },
  {
    "name": "itty-pts-3",
    "windows": 1,
    "created": "2026-03-28T11:15:00-07:00",
    "attached": false,
    "lastPaneCommand": "claude",
    "lastPanePath": "/home/user/work"
  }
]
```

### GET /sessions/{name}

Returns detailed information about a specific session including windows and panes.

**Response:**
```json
{
  "name": "itty-pts-1",
  "windows": 2,
  "created": "2026-03-28T10:30:00-07:00",
  "attached": true,
  "lastPaneCommand": "nvim",
  "lastPanePath": "/home/user/project",
  "windowList": [
    {
      "index": 1,
      "name": "editor",
      "active": true,
      "panes": [
        {
          "id": "%0",
          "index": 1,
          "active": true,
          "command": "nvim",
          "path": "/home/user/project",
          "width": 120,
          "height": 40
        }
      ]
    },
    {
      "index": 2,
      "name": "shell",
      "active": false,
      "panes": [
        {
          "id": "%1",
          "index": 1,
          "active": true,
          "command": "bash",
          "path": "/home/user/project",
          "width": 120,
          "height": 40
        }
      ]
    }
  ]
}
```

### GET /sessions/{name}/content

Captures the visible content of the active pane in a session.

**Response:**
```json
{
  "content": "~ ❯ ls\nREADME.md  src/  tests/\n~ ❯ "
}
```

### GET /config

Returns current daemon configuration.

### PUT /config/auto

Toggle auto-tmux shell wrapping.

**Request:**
```json
{
  "enabled": true
}
```

### GET /windows

Lists open terminal windows on the desktop (platform-specific).

**Response:**
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

### WS /stream

WebSocket endpoint for real-time session updates. *(Planned)*

### POST /notify/register

Register an APNs device token for push notifications. *(Planned)*
