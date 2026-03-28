# ADR 004: Tailscale-First Networking

## Status
Accepted

## Context
Users need to connect their phone to their desktop. Options: raw SSH over the internet (port forwarding), VPN, Tailscale, or cloud relay.

## Decision
Tailscale as the primary networking layer, with manual SSH (any hostname/IP) as a fallback.

## Rationale
- **Zero config**: Tailscale handles NAT traversal, encryption, and device discovery automatically
- **Private network**: Daemon is only accessible on the user's tailnet — no public exposure
- **Device discovery**: Tailscale REST API (`GET /api/v2/tailnet/{}/devices`) lists all machines with status
- **Tailscale SSH**: Can authenticate without SSH keys using tailnet identity
- **Tailscale Serve**: Exposes daemon on tailnet with auto-HTTPS — `tailscale serve 8080`
- **Already installed**: Many developers already use Tailscale

## No Embeddable SDK
Tailscale has no iOS SDK for embedding. The app relies on the system Tailscale VPN being active. This is actually simpler — install Tailscale once, all apps benefit.

## Manual SSH Fallback
Users can also add machines by hostname/IP for non-Tailscale setups. Standard SSH key/password auth.

## Consequences
- Requires Tailscale app installed on iOS (separate download)
- iOS allows only one VPN at a time — if user has another VPN, Tailscale may conflict
- VPN status detected via NEVPNManager (read-only, no special entitlement)
