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
            echo "${RED}Unsupported operating system: $OS${NC}"
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
            echo "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac
}

# Get latest release version
get_latest_version() {
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        printf "%bFailed to get latest version%b\n" "$RED" "$NC"
        exit 1
    fi
    
    printf "%bLatest version: %s%b\n" "$GREEN" "$VERSION" "$NC"
}

# Download binary
download_binary() {
    BINARY_URL="https://github.com/$REPO/releases/download/$VERSION/keyrafted-${OS}-${ARCH}"
    
    if [ "$OS" = "windows" ]; then
        BINARY_URL="${BINARY_URL}.exe"
    fi
    
    printf "%bDownloading from: %s%b\n" "$YELLOW" "$BINARY_URL" "$NC"
    
    TMP_FILE=$(mktemp)
    
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$BINARY_URL" -o "$TMP_FILE"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$BINARY_URL" -O "$TMP_FILE"
    else
        printf "%bError: curl or wget is required%b\n" "$RED" "$NC"
        exit 1
    fi
    
    if [ ! -s "$TMP_FILE" ]; then
        printf "%bFailed to download binary%b\n" "$RED" "$NC"
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
        echo "${YELLOW}Installing to $INSTALL_DIR (requires sudo)${NC}"
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    echo "${GREEN}✓ Installed $BINARY_NAME to $INSTALL_DIR${NC}"
}

# Verify installation
verify_installation() {
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        echo "${YELLOW}Warning: $BINARY_NAME not found in PATH${NC}"
        echo "Add $INSTALL_DIR to your PATH or run: export PATH=\"$INSTALL_DIR:\$PATH\""
        return
    fi
    
    VERSION_OUTPUT=$("$BINARY_NAME" --version 2>&1 || "$BINARY_NAME" --help 2>&1 | head -1)
    echo "${GREEN}✓ Installation verified${NC}"
    echo "$VERSION_OUTPUT"
}

# Main installation
main() {
    printf "%bKeyraft Installation Script%b\n\n" "$GREEN" "$NC"
    
    detect_platform
    printf "Detected platform: %b%s/%s%b\n" "$YELLOW" "$OS" "$ARCH" "$NC"
    
    get_latest_version
    
    TMP_FILE=$(download_binary)
    
    install_binary "$TMP_FILE"
    
    verify_installation
    
    printf "\n%bInstallation complete!%b\n\n" "$GREEN" "$NC"
    printf "Quick start:\n"
    printf "  1. Initialize: %bkeyrafted init --data-dir ./data%b\n" "$YELLOW" "$NC"
    printf "  2. Start:      %bexport KEYRAFT_MASTER_KEY=\$(openssl rand -base64 32)%b\n" "$YELLOW" "$NC"
    printf "                 %bkeyrafted start --data-dir ./data%b\n" "$YELLOW" "$NC"
    printf "\nDocumentation: %bhttps://github.com/%s%b\n" "$YELLOW" "$REPO" "$NC"
}

main

