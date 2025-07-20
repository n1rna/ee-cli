#!/bin/bash

# menv installer script
# Usage: curl -sSfL https://raw.githubusercontent.com/n1rna/menv/main/install.sh | sh

set -e

# Configuration
REPO="n1rna/menv"
BINARY_NAME="menv"
VERSION=""  # Will be set to latest if empty

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
debug() { [[ ${DEBUG:-} ]] && echo -e "${BLUE}[DEBUG]${NC} $1" || true; }

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
    if [[ "$os" == "windows" ]]; then
        BINARY_SUFFIX=".exe"
    else
        BINARY_SUFFIX=""
    fi
}

# Get the latest version from GitHub API
get_latest_version() {
    if [[ -n "$VERSION" ]]; then
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
    
    if [[ -z "$VERSION" ]]; then
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

# Download and install
install_binary() {
    local tmp_dir="/tmp/menv-install-$$"
    local binary_name="${BINARY_NAME}-${PLATFORM}${BINARY_SUFFIX}"
    local download_url="https://github.com/$REPO/releases/download/$VERSION/$binary_name"
    
    log "Downloading menv $VERSION for $PLATFORM..."
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
    
    # Determine install location
    local install_dir
    if [[ -w "/usr/local/bin" ]]; then
        install_dir="/usr/local/bin"
    elif [[ -d "$HOME/.local/bin" ]]; then
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
    elif [[ -d "$HOME/bin" ]]; then
        install_dir="$HOME/bin"
    else
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
        warn "Created directory $install_dir - make sure it's in your PATH"
    fi
    
    # Install binary
    log "Installing to $install_dir/$BINARY_NAME$BINARY_SUFFIX..."
    
    if ! cp "$tmp_dir/$BINARY_NAME$BINARY_SUFFIX" "$install_dir/$BINARY_NAME$BINARY_SUFFIX"; then
        error "Failed to install binary to $install_dir"
        warn "You may need to run with sudo or choose a different install location"
        exit 1
    fi
    
    # Verify installation
    if "$install_dir/$BINARY_NAME$BINARY_SUFFIX" version >/dev/null 2>&1; then
        log "✅ menv $VERSION installed successfully!"
        log "📍 Location: $install_dir/$BINARY_NAME$BINARY_SUFFIX"
        
        # Check if binary is in PATH
        if command -v "$BINARY_NAME" >/dev/null 2>&1; then
            log "🎉 You can now use 'menv' from anywhere!"
        else
            warn "⚠️  $install_dir is not in your PATH"
            warn "Add this to your shell profile:"
            warn "export PATH=\"$install_dir:\$PATH\""
        fi
        
        log ""
        log "Get started with: menv --help"
        log "Documentation: https://github.com/$REPO"
    else
        error "Installation verification failed"
        exit 1
    fi
}

# Download checksums and verify
verify_checksum() {
    if [[ "${SKIP_CHECKSUM:-}" == "true" ]]; then
        warn "Skipping checksum verification"
        return
    fi
    
    local tmp_dir="/tmp/menv-install-$$"
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
        
        if [[ "$expected_checksum" != "$actual_checksum" ]]; then
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
    log "🚀 Installing menv CLI tool..."
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
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
menv installer script

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