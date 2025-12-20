#!/bin/sh
# Keyraft Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/keyraft/keyrafted/main/install.sh | sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="keyraft/keyrafted"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="keyrafted"

# Detect OS and Architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$OS" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            OS="windows"
            BINARY_NAME="keyrafted.exe"
            ;;
        *)
            printf "%bUnsupported operating system: %s%b\n" "$RED" "$OS" "$NC" >&2
            exit 1
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            printf "%bUnsupported architecture: %s%b\n" "$RED" "$ARCH" "$NC" >&2
            exit 1
            ;;
    esac
}

# Get latest release version
get_latest_version() {
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        printf "%bFailed to get latest version%b\n" "$RED" "$NC" >&2
        exit 1
    fi
    
    printf "%bLatest version: %s%b\n" "$GREEN" "$VERSION" "$NC" >&2
}

# Download binary
download_binary() {
    BINARY_URL="https://github.com/$REPO/releases/download/$VERSION/keyrafted-${OS}-${ARCH}"
    
    if [ "$OS" = "windows" ]; then
        BINARY_URL="${BINARY_URL}.exe"
    fi
    
    printf "%bDownloading from: %s%b\n" "$YELLOW" "$BINARY_URL" "$NC" >&2
    
    TMP_FILE=$(mktemp)
    
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$BINARY_URL" -o "$TMP_FILE"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$BINARY_URL" -O "$TMP_FILE"
    else
        printf "%bError: curl or wget is required%b\n" "$RED" "$NC" >&2
        exit 1
    fi
    
    if [ ! -s "$TMP_FILE" ]; then
        printf "%bFailed to download binary%b\n" "$RED" "$NC" >&2
        rm -f "$TMP_FILE"
        exit 1
    fi
    
    echo "$TMP_FILE"
}

# Install binary
install_binary() {
    TMP_FILE=$1
    
    chmod +x "$TMP_FILE"
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    else
        printf "%bInstalling to %s (requires sudo)%b\n" "$YELLOW" "$INSTALL_DIR" "$NC" >&2
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    printf "%b✓ Installed %s to %s%b\n" "$GREEN" "$BINARY_NAME" "$INSTALL_DIR" "$NC" >&2
}

# Verify installation
verify_installation() {
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        printf "%bWarning: %s not found in PATH%b\n" "$YELLOW" "$BINARY_NAME" "$NC" >&2
        printf "Add %s to your PATH or run: export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR" "$INSTALL_DIR" >&2
        return
    fi
    
    # Test that binary works by running help
    if "$BINARY_NAME" --help >/dev/null 2>&1; then
        printf "%b✓ Installation verified%b\n" "$GREEN" "$NC" >&2
    else
        printf "%bWarning: Binary installed but may not be working correctly%b\n" "$YELLOW" "$NC" >&2
    fi
}

# Main installation
main() {
    printf "%bKeyraft Installation Script%b\n\n" "$GREEN" "$NC" >&2
    
    detect_platform
    printf "Detected platform: %b%s/%s%b\n" "$YELLOW" "$OS" "$ARCH" "$NC" >&2
    
    get_latest_version
    
    TMP_FILE=$(download_binary)
    
    install_binary "$TMP_FILE"
    
    verify_installation
    
    printf "\n%bInstallation complete!%b\n\n" "$GREEN" "$NC" >&2
    printf "Quick start:\n" >&2
    printf "  1. Initialize: %bkeyrafted init --data-dir ./data%b\n" "$YELLOW" "$NC" >&2
    printf "  2. Start:      %bexport KEYRAFT_MASTER_KEY=\$(openssl rand -base64 32)%b\n" "$YELLOW" "$NC" >&2
    printf "                 %bkeyrafted start --data-dir ./data%b\n" "$YELLOW" "$NC" >&2
    printf "\nDocumentation: %bhttps://github.com/%s%b\n" "$YELLOW" "$REPO" "$NC" >&2
}

main

