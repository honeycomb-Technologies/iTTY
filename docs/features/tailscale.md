# Tailscale

iTTY uses Tailscale as the default network path between the iPhone app and the desktop daemon.

## What Exists Today

- the daemon exposes HTTP on its configured `listen_addr` locally
- the installer configures `tailscale serve --bg 8080` when Tailscale is already installed and connected
- daemon startup re-applies `tailscale serve --bg <listen-port>` when `tailscale_serve = true`
- `itty status` reports whether the Tailscale CLI is installed, whether it is live, and the current tailnet hostname when available

## What Is Optional

Tailscale is not required for local daemon operation.

If the CLI is missing, not logged in, or `tailscale serve` fails, the daemon still starts and serves locally. The failure is logged so the user can correct it later.

## What Does Not Exist Yet

- app-driven toggles for Tailscale Serve
- TLS or auth management outside the Tailscale Serve model
- daemon-managed installation of the Tailscale client itself
