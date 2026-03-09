#!/bin/bash

# ee installer script
# Usage: curl -sSfL https://raw.githubusercontent.com/n1rna/ee-cli/main/install.sh | sh

set -e

# Configuration
REPO="n1rna/ee-cli"
BINARY_NAME="ee"
VERSION=""  # Will be set to latest if empty

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log() { printf "${GREEN}[INFO]${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; }
error() { printf "${RED}[ERROR]${NC} %s\n" "$1" >&2; }
debug() { [ "${DEBUG:-}" ] && printf "${BLUE}[DEBUG]${NC} %s\n" "$1" || true; }

# Detect OS and architecture
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*) os="linux" ;;
        Darwin*) os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *) error "Unsupported operating system: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        armv7*) arch="arm" ;;
        i386|i686) arch="386" ;;
        *) error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    PLATFORM="${os}-${arch}"
    debug "Detected platform: $PLATFORM"

    # Set binary suffix for Windows
    if [ "$os" = "windows" ]; then
        BINARY_SUFFIX=".exe"
    else
        BINARY_SUFFIX=""
    fi
}

# Get the latest version from GitHub API
get_latest_version() {
    if [ -n "$VERSION" ]; then
        debug "Using specified version: $VERSION"
        return
    fi

    log "Fetching latest version..."

    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi

    if [ -z "$VERSION" ]; then
        error "Failed to get the latest version"
        exit 1
    fi

    debug "Latest version: $VERSION"
}

# Check if version exists
check_version_exists() {
    local url="https://api.github.com/repos/$REPO/releases/tags/$VERSION"

    debug "Checking if version $VERSION exists..."

    if command -v curl >/dev/null 2>&1; then
        if ! curl -sf "$url" >/dev/null; then
            error "Version $VERSION not found"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q --spider "$url"; then
            error "Version $VERSION not found"
            exit 1
        fi
    fi
}

# Check if a directory is in the current PATH
is_in_path() {
    case ":$PATH:" in
        *":$1:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Find the best user-writable install directory
# Priority: user-writable directories already in PATH, then well-known user dirs
find_install_dir() {
    # Well-known user-level bin directories, in order of preference
    local candidates="
        $HOME/.local/bin
        $HOME/bin
        $HOME/.cargo/bin
        $HOME/go/bin
        $HOME/.local/share/bin
    "

    # 1. Check if any well-known user directory is already in PATH and writable
    for dir in $candidates; do
        if is_in_path "$dir" && [ -d "$dir" ] && [ -w "$dir" ]; then
            debug "Found user-writable directory in PATH: $dir"
            echo "$dir"
            return
        fi
    done

    # 2. Scan PATH for any other user-writable directory (under $HOME)
    local IFS=':'
    for dir in $PATH; do
        case "$dir" in
            "$HOME"*)
                if [ -d "$dir" ] && [ -w "$dir" ]; then
                    debug "Found user-writable directory in PATH: $dir"
                    echo "$dir"
                    return
                fi
                ;;
        esac
    done
    unset IFS

    # 3. Check /usr/local/bin if writable (common on macOS)
    if [ -w "/usr/local/bin" ]; then
        debug "Using writable /usr/local/bin"
        echo "/usr/local/bin"
        return
    fi

    # 4. Create ~/.local/bin (XDG standard, most shells source it)
    local fallback="$HOME/.local/bin"
    mkdir -p "$fallback"

    # Check if it's already in PATH after creation
    if is_in_path "$fallback"; then
        debug "Created $fallback (already in PATH)"
    else
        debug "Created $fallback (not yet in PATH)"
    fi

    echo "$fallback"
}

# Detect the user's shell profile file
detect_shell_profile() {
    local shell_name
    shell_name=$(basename "${SHELL:-/bin/sh}")

    case "$shell_name" in
        zsh)
            if [ -f "$HOME/.zshrc" ]; then
                echo "$HOME/.zshrc"
            else
                echo "$HOME/.zprofile"
            fi
            ;;
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                echo "$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.profile"
            fi
            ;;
        fish)
            echo "$HOME/.config/fish/config.fish"
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# Suggest how to add a directory to PATH
suggest_path_update() {
    local dir="$1"
    local profile
    profile=$(detect_shell_profile)
    local shell_name
    shell_name=$(basename "${SHELL:-/bin/sh}")

    warn "Add $dir to your PATH by running:"
    if [ "$shell_name" = "fish" ]; then
        warn "  fish_add_path $dir"
    else
        warn "  echo 'export PATH=\"$dir:\$PATH\"' >> $profile"
    fi
    warn "Then restart your shell or run:"
    if [ "$shell_name" = "fish" ]; then
        warn "  source $profile"
    else
        warn "  source $profile"
    fi
}

# Download and install
install_binary() {
    local tmp_dir="/tmp/ee-install-$$"
    local binary_name="${BINARY_NAME}-${PLATFORM}${BINARY_SUFFIX}"
    local download_url="https://github.com/$REPO/releases/download/$VERSION/$binary_name"

    log "Downloading ee $VERSION for $PLATFORM..."
    debug "Download URL: $download_url"

    # Create temporary directory
    mkdir -p "$tmp_dir"
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download binary
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sL "$download_url" -o "$tmp_dir/$BINARY_NAME$BINARY_SUFFIX"; then
            error "Failed to download $binary_name"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$tmp_dir/$BINARY_NAME$BINARY_SUFFIX"; then
            error "Failed to download $binary_name"
            exit 1
        fi
    fi

    # Make binary executable
    chmod +x "$tmp_dir/$BINARY_NAME$BINARY_SUFFIX"

    # Determine install location — prefer user-writable directories already in PATH
    local install_dir
    install_dir=$(find_install_dir)

    # Install binary
    log "Installing to $install_dir/$BINARY_NAME$BINARY_SUFFIX..."

    if ! cp "$tmp_dir/$BINARY_NAME$BINARY_SUFFIX" "$install_dir/$BINARY_NAME$BINARY_SUFFIX"; then
        error "Failed to install binary to $install_dir"
        warn "You may need to run with sudo or choose a different install location"
        exit 1
    fi

    # Verify installation
    if "$install_dir/$BINARY_NAME$BINARY_SUFFIX" --version >/dev/null 2>&1; then
        log "ee $VERSION installed successfully!"
        log "Location: $install_dir/$BINARY_NAME$BINARY_SUFFIX"

        # Check if binary is in PATH
        if command -v "$BINARY_NAME" >/dev/null 2>&1; then
            log "You can now use 'ee' from anywhere!"
        else
            warn "$install_dir is not in your PATH"
            suggest_path_update "$install_dir"
        fi

        log ""
        log "Get started with: ee --help"
        log "Documentation: https://github.com/$REPO"
    else
        error "Installation verification failed"
        exit 1
    fi
}

# Download checksums and verify
verify_checksum() {
    if [ "${SKIP_CHECKSUM:-}" = "true" ]; then
        warn "Skipping checksum verification"
        return
    fi

    local tmp_dir="/tmp/ee-install-$$"
    local binary_name="${BINARY_NAME}-${PLATFORM}${BINARY_SUFFIX}"
    local checksums_url="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"

    log "Verifying checksum..."

    # Download checksums
    if command -v curl >/dev/null 2>&1; then
        curl -sL "$checksums_url" -o "$tmp_dir/checksums.txt"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$checksums_url" -O "$tmp_dir/checksums.txt"
    else
        warn "Cannot verify checksum: neither curl nor wget available"
        return
    fi

    # Verify checksum
    if command -v sha256sum >/dev/null 2>&1; then
        local expected_checksum=$(grep "$binary_name" "$tmp_dir/checksums.txt" | cut -d' ' -f1)
        local actual_checksum=$(sha256sum "$tmp_dir/$BINARY_NAME$BINARY_SUFFIX" | cut -d' ' -f1)

        if [ "$expected_checksum" != "$actual_checksum" ]; then
            error "Checksum verification failed!"
            error "Expected: $expected_checksum"
            error "Actual: $actual_checksum"
            exit 1
        fi

        log "✅ Checksum verified"
    else
        warn "sha256sum not available - skipping checksum verification"
    fi
}

# Main installation flow
main() {
    log "🚀 Installing ee CLI tool..."

    # Parse arguments
    while [ $# -gt 0 ]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --debug)
                DEBUG=true
                shift
                ;;
            --skip-checksum)
                SKIP_CHECKSUM=true
                shift
                ;;
            --help|-h)
                cat << EOF
ee installer script

Usage: $0 [options]

Options:
    --version VERSION    Install specific version (default: latest)
    --debug             Enable debug output
    --skip-checksum     Skip checksum verification
    --help, -h          Show this help message

Examples:
    $0                          # Install latest version
    $0 --version v1.0.0         # Install specific version
    $0 --debug                  # Install with debug output

EOF
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                error "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    detect_platform
    get_latest_version
    check_version_exists
    install_binary
    verify_checksum
}

# Run main function
main "$@"