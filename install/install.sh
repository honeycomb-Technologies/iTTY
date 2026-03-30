#!/usr/bin/env bash
# iTTY Daemon Installer
# Usage: curl -fsSL https://github.com/honeycomb-Technologies/iTTY/releases/latest/download/install.sh | bash
#
# This script:
# 1. Detects the OS and architecture
# 2. Installs tmux if not present
# 3. Downloads the iTTY daemon binary
# 4. Configures the shell for auto-tmux wrapping
# 5. Installs a system service (systemd or launchd)
# 6. Starts the daemon
# 7. Optionally configures Tailscale Serve

set -euo pipefail

REPO="honeycomb-Technologies/iTTY"
INSTALL_DIR="$HOME/.local/bin"
VERSION="${ITTY_VERSION:-latest}"
SCRIPT_DIR=""
OS=""
ARCH=""
BINARY_NAME=""

if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

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

release_asset_url() {
    local asset="$1"

    if [ "$VERSION" = "latest" ]; then
        printf 'https://github.com/%s/releases/latest/download/%s\n' "$REPO" "$asset"
        return
    fi

    printf 'https://github.com/%s/releases/download/%s/%s\n' "$REPO" "$VERSION" "$asset"
}

download_to() {
    local url="$1"
    local dest="$2"

    if command -v curl &>/dev/null; then
        curl --fail --silent --show-error --location --retry 3 --connect-timeout 10 "$url" -o "$dest"
    elif command -v wget &>/dev/null; then
        wget --quiet "$url" -O "$dest"
    else
        fail "neither curl nor wget found"
    fi
}

find_local_binary() {
    local candidate

    for candidate in \
        "${SCRIPT_DIR}/../daemon/bin/${BINARY_NAME}" \
        "${SCRIPT_DIR}/../daemon/bin/itty"; do
        if [ -f "$candidate" ]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done

    return 1
}

find_local_install_asset() {
    local asset="$1"
    local candidate

    for candidate in \
        "${SCRIPT_DIR}/${asset}" \
        "${SCRIPT_DIR}/install/${asset}"; do
        if [ -f "$candidate" ]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done

    return 1
}

copy_or_download_install_asset() {
    local asset="$1"
    local dest="$2"
    local local_asset

    if local_asset="$(find_local_install_asset "$asset")"; then
        cp "$local_asset" "$dest"
        return
    fi

    download_to "$(release_asset_url "$asset")" "$dest"
}

escape_sed_replacement() {
    printf '%s' "$1" | sed 's/[\\/&]/\\&/g'
}

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

    BINARY_NAME="itty-${OS}-${ARCH}"
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
    local local_binary

    mkdir -p "$INSTALL_DIR"

    if local_binary="$(find_local_binary)"; then
        info "installing iTTY daemon from local build: ${local_binary}"
        cp "$local_binary" "${INSTALL_DIR}/itty"
    else
        info "downloading iTTY daemon..."
        download_to "$(release_asset_url "$BINARY_NAME")" "${INSTALL_DIR}/itty" || fail "download failed for $(release_asset_url "$BINARY_NAME")"
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
    local service_dir="$HOME/.config/systemd/user"
    local service_path="${service_dir}/itty-daemon.service"
    local tmp_asset

    if ! command -v systemctl &>/dev/null; then
        warn "systemctl not found — skipping systemd service install"
        return
    fi

    mkdir -p "$service_dir"
    tmp_asset="$(mktemp)"
    copy_or_download_install_asset "itty-daemon.service" "$tmp_asset"
    install -m 0644 "$tmp_asset" "$service_path"
    rm -f "$tmp_asset"

    systemctl --user daemon-reload
    systemctl --user enable --now itty-daemon.service
    ok "systemd user service installed and started"
}

install_launchd_service() {
    local plist_dir="$HOME/Library/LaunchAgents"
    local log_dir="$HOME/Library/Logs/iTTY"
    local plist_path="${plist_dir}/com.honeycomb.itty.plist"
    local tmp_asset

    mkdir -p "$plist_dir" "$log_dir"
    tmp_asset="$(mktemp)"
    copy_or_download_install_asset "com.honeycomb.itty.plist" "$tmp_asset"
    sed \
        -e "s|__ITTY_BIN__|$(escape_sed_replacement "${INSTALL_DIR}/itty")|g" \
        -e "s|__ITTY_LOG__|$(escape_sed_replacement "${log_dir}/daemon.log")|g" \
        -e "s|__ITTY_PATH__|$(escape_sed_replacement "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin")|g" \
        "$tmp_asset" > "$plist_path"
    rm -f "$tmp_asset"

    launchctl unload "$plist_path" 2>/dev/null || true
    launchctl load "$plist_path"
    ok "launchd agent installed and started"
}

# --- Configure Tailscale Serve ---

configure_tailscale() {
    if ! command -v tailscale &>/dev/null; then
        warn "tailscale not found — skipping Tailscale Serve setup"
        warn "install Tailscale for zero-config phone access: https://tailscale.com/download"
        return
    fi

    if ! tailscale status --json &>/dev/null; then
        warn "tailscale not connected — skipping Tailscale Serve setup"
        warn "connect with: tailscale up"
        return
    fi

    info "configuring Tailscale Serve..."
    if tailscale serve --bg 3420 &>/dev/null; then
        ok "Tailscale Serve configured for local port 3420"
    else
        warn "tailscale serve failed — you can configure it later with: tailscale serve --bg 3420"
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
    ok "  ${INSTALL_DIR}/itty status"
    ok "  ${INSTALL_DIR}/itty sessions"
    ok "  ${INSTALL_DIR}/itty auto off"
    ok "═══════════════════════════════════════"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
