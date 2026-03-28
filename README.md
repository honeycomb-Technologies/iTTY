# iTTY

**Persistent remote terminal for iPhone** — see your desktop terminals from your phone, pick up where you left off.

## What It Does

1. **Install the daemon** on your desktop (one command)
2. **Install iTTY** from the App Store on your phone
3. **Open iTTY** → see all your open terminal sessions → tap one → you're in

Sessions persist when you close the app. Go to your desktop, same state. Go back to your phone, same state. Always in sync.

## How It Works

- **iTTY iOS App**: Ghostty-powered terminal with Metal GPU rendering at 120fps
- **iTTY Daemon**: Lightweight Go binary on your desktop that auto-configures tmux for session persistence
- **Tailscale**: Zero-config private networking between your devices

Your desktop terminal doesn't matter — Ghostty, Alacritty, Kitty, the default macOS Terminal — anything works. tmux runs invisibly under the hood, keeping your sessions alive.

## Quick Start

### Desktop
```bash
curl -fsSL https://itty.app/install | bash
```

### iPhone
Download iTTY from the App Store. *(Coming soon)*

## Development

See [CLAUDE.md](CLAUDE.md) for build commands and development guide.
See [docs/](docs/) for full documentation.

## Architecture

```
Desktop (any terminal + tmux + iTTY daemon)
    ↕ Tailscale (private network)
iPhone (iTTY app with Ghostty rendering)
```

## License

MIT

---

Built by [honeycomb.Technologies](https://github.com/honeycomb-Technologies)
