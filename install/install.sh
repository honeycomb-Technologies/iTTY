#!/usr/bin/env bash
# iTTY Daemon Installer
# Usage: curl -fsSL https://itty.app/install | bash
#
# This script:
# 1. Detects the OS and architecture
# 2. Installs tmux if not present
# 3. Downloads the iTTY daemon binary
# 4. Configures the shell for auto-tmux wrapping
# 5. Installs a system service (systemd or launchd)
# 6. Starts the daemon
# 7. Configures Tailscale Serve (if available)

set -euo pipefail

REPO="honeycomb-Technologies/iTTY"
INSTALL_DIR="$HOME/.local/bin"
VERSION="${ITTY_VERSION:-latest}"

# Colors (if terminal supports them)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' NC=''
fi

info()  { echo -e "${BLUE}[iTTY]${NC} $*"; }
ok()    { echo -e "${GREEN}[iTTY]${NC} $*"; }
warn()  { echo -e "${YELLOW}[iTTY]${NC} $*"; }
fail()  { echo -e "${RED}[iTTY]${NC} $*" >&2; exit 1; }

# --- Detect OS and architecture ---

detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) fail "unsupported architecture: $ARCH" ;;
    esac

    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      fail "unsupported OS: $OS" ;;
    esac

    info "detected platform: ${OS}/${ARCH}"
}

# --- Install tmux if missing ---

install_tmux() {
    if command -v tmux &>/dev/null; then
        ok "tmux already installed: $(tmux -V)"
        return
    fi

    info "installing tmux..."
    case "$OS" in
        linux)
            if command -v apt-get &>/dev/null; then
                sudo apt-get update -qq && sudo apt-get install -y -qq tmux
            elif command -v dnf &>/dev/null; then
                sudo dnf install -y tmux
            elif command -v pacman &>/dev/null; then
                sudo pacman -S --noconfirm tmux
            elif command -v apk &>/dev/null; then
                sudo apk add tmux
            else
                fail "cannot detect package manager — install tmux manually"
            fi
            ;;
        darwin)
            if command -v brew &>/dev/null; then
                brew install tmux
            else
                fail "homebrew not found — install tmux with: brew install tmux"
            fi
            ;;
    esac

    if ! command -v tmux &>/dev/null; then
        fail "tmux installation failed"
    fi
    ok "tmux installed: $(tmux -V)"
}

# --- Download daemon binary ---

download_daemon() {
    mkdir -p "$INSTALL_DIR"

    BINARY_NAME="itty-${OS}-${ARCH}"
    if [ "$VERSION" = "latest" ]; then
        DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"
    else
        DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
    fi

    info "downloading iTTY daemon..."
    if command -v curl &>/dev/null; then
        curl -fsSL "$DOWNLOAD_URL" -o "${INSTALL_DIR}/itty" || fail "download failed from ${DOWNLOAD_URL}"
    elif command -v wget &>/dev/null; then
        wget -q "$DOWNLOAD_URL" -O "${INSTALL_DIR}/itty" || fail "download failed from ${DOWNLOAD_URL}"
    else
        fail "neither curl nor wget found"
    fi

    chmod +x "${INSTALL_DIR}/itty"
    ok "daemon installed to ${INSTALL_DIR}/itty"
}

# --- Configure shell auto-wrap ---

configure_shell() {
    info "configuring shell for auto-tmux..."
    "${INSTALL_DIR}/itty" auto on
    ok "shell configured — new terminals will auto-wrap in tmux"
}

# --- Install system service ---

install_service() {
    case "$OS" in
        linux)
            install_systemd_service
            ;;
        darwin)
            install_launchd_service
            ;;
    esac
}

install_systemd_service() {
    SERVICE_DIR="$HOME/.config/systemd/user"
    mkdir -p "$SERVICE_DIR"

    cat > "${SERVICE_DIR}/itty-daemon.service" << 'UNIT'
[Unit]
Description=iTTY Daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%h/.local/bin/itty start
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
UNIT

    systemctl --user daemon-reload
    systemctl --user enable itty-daemon.service
    systemctl --user start itty-daemon.service
    ok "systemd user service installed and started"
}

install_launchd_service() {
    PLIST_DIR="$HOME/Library/LaunchAgents"
    mkdir -p "$PLIST_DIR"

    ITTY_PATH="${INSTALL_DIR}/itty"
    cat > "${PLIST_DIR}/com.honeycomb.itty.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.honeycomb.itty</string>
    <key>ProgramArguments</key>
    <array>
        <string>${ITTY_PATH}</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/itty-daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/itty-daemon.log</string>
</dict>
</plist>
PLIST

    launchctl load "${PLIST_DIR}/com.honeycomb.itty.plist" 2>/dev/null || true
    ok "launchd service installed and started"
}

# --- Configure Tailscale Serve ---

configure_tailscale() {
    if ! command -v tailscale &>/dev/null; then
        warn "tailscale not found — skipping Tailscale Serve setup"
        warn "install Tailscale for zero-config phone access: https://tailscale.com/download"
        return
    fi

    if ! tailscale status &>/dev/null; then
        warn "tailscale not connected — skipping Tailscale Serve setup"
        warn "connect with: tailscale up"
        return
    fi

    info "configuring Tailscale Serve..."
    if tailscale serve --bg 8080 &>/dev/null; then
        HOSTNAME=$(tailscale status --self --json 2>/dev/null | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')
        ok "Tailscale Serve configured — daemon accessible at https://${HOSTNAME:-your-machine}:8080"
    else
        warn "tailscale serve failed — you can configure it later with: tailscale serve --bg 8080"
    fi
}

# --- Main ---

main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║        iTTY Daemon Installer         ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""

    detect_platform
    install_tmux
    download_daemon
    configure_shell
    install_service
    configure_tailscale

    echo ""
    ok "═══════════════════════════════════════"
    ok "iTTY daemon installed and running!"
    ok ""
    ok "Next steps:"
    ok "  1. Install iTTY on your iPhone"
    ok "  2. Install Tailscale on both devices"
    ok "  3. Open iTTY — your terminals are waiting"
    ok ""
    ok "Commands:"
    ok "  itty status     Show daemon status"
    ok "  itty sessions   List tmux sessions"
    ok "  itty auto off   Disable auto-tmux wrapping"
    ok "═══════════════════════════════════════"
}

main "$@"
