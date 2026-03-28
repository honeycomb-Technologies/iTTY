# Daemon Install

## One-Command Install

Install the latest released daemon with:

```bash
curl -fsSL https://github.com/honeycomb-Technologies/iTTY/releases/latest/download/install.sh | bash
```

The installer:

- detects the current `linux|darwin` and `amd64|arm64` platform
- installs `tmux` with the platform package manager if needed
- installs `itty` to `~/.local/bin/itty`
- enables shell auto-wrap with `itty auto on`
- installs and starts a per-user service
- configures `tailscale serve --bg 8080` when Tailscale is installed and connected

Pin a specific release with:

```bash
ITTY_VERSION=v0.1.0 curl -fsSL https://github.com/honeycomb-Technologies/iTTY/releases/download/v0.1.0/install.sh | bash
```

## Manual Build From Source

```bash
git clone https://github.com/honeycomb-Technologies/iTTY.git
cd iTTY/daemon
make build
./bin/itty start
```

To install from source into your Go bin directory:

```bash
cd daemon
make install
```

## Services

On Linux, the installer installs `~/.config/systemd/user/itty-daemon.service` and starts it with:

```bash
systemctl --user enable --now itty-daemon.service
```

On macOS, the installer installs `~/Library/LaunchAgents/com.honeycomb.itty.plist` and loads it with `launchctl`.

The systemd user service keeps `ProtectSystem=strict` and `NoNewPrivileges=yes`, but it does not mount `HOME` read-only because the daemon may create or update:

- `~/.config/itty/config.toml`
- `~/.bashrc`
- `~/.zshrc`
- `~/.config/fish/config.fish`

## Tailscale

If `tailscale` is installed and connected, the installer configures:

```bash
tailscale serve --bg 8080
```

At daemon startup, `tailscale_serve = true` also causes the daemon to re-assert that serve configuration when Tailscale is available.

## Verify

```bash
~/.local/bin/itty status
~/.local/bin/itty sessions
```
