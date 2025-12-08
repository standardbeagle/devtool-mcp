#!/bin/bash
#
# devtool-mcp installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/standardbeagle/devtool-mcp/main/install.sh | bash
#
# Or with a specific version:
#   curl -fsSL https://raw.githubusercontent.com/standardbeagle/devtool-mcp/main/install.sh | bash -s -- --version 0.3.0
#

set -e

VERSION="${VERSION:-0.3.0}"
REPO="standardbeagle/devtool-mcp"
BINARY_NAME="devtool-mcp"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "devtool-mcp installer"
            echo ""
            echo "Usage:"
            echo "  curl -fsSL https://raw.githubusercontent.com/standardbeagle/devtool-mcp/main/install.sh | bash"
            echo ""
            echo "Options:"
            echo "  --version VERSION    Install specific version (default: $VERSION)"
            echo "  --install-dir DIR    Install to specific directory (default: ~/.local/bin)"
            echo "  -h, --help           Show this help message"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        CYGWIN*|MINGW*|MSYS*) echo "windows";;
        *)          error "Unsupported operating system: $(uname -s)";;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64";;
        arm64|aarch64)  echo "arm64";;
        *)              error "Unsupported architecture: $(uname -m)";;
    esac
}

# Get download URL
get_download_url() {
    local os="$1"
    local arch="$2"
    local ext=""

    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    echo "https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY_NAME}-${os}-${arch}${ext}"
}

# Download file
download() {
    local url="$1"
    local dest="$2"

    if command -v curl &> /dev/null; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget &> /dev/null; then
        wget -q -O "$dest" "$url"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Main installation
main() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    local ext=""

    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    info "Installing devtool-mcp v${VERSION}"
    info "  OS: $os"
    info "  Architecture: $arch"
    info "  Install directory: $INSTALL_DIR"

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Get download URL
    local url=$(get_download_url "$os" "$arch")
    local binary_path="${INSTALL_DIR}/${BINARY_NAME}${ext}"

    info "Downloading from: $url"

    # Download binary
    if ! download "$url" "$binary_path"; then
        error "Failed to download binary from $url"
    fi

    # Make executable
    chmod +x "$binary_path"

    success "Installed devtool-mcp to $binary_path"

    # Check if install directory is in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        warn "Installation directory is not in your PATH"
        echo ""
        echo "Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
    fi

    # Verify installation
    if "$binary_path" --version &> /dev/null; then
        success "Installation verified: $($binary_path --version)"
    else
        warn "Could not verify installation. Binary may require additional setup."
    fi

    echo ""
    info "To use with Claude Code or other MCP clients, add to your config:"
    echo ""
    echo '  {'
    echo '    "mcpServers": {'
    echo '      "devtool": {'
    echo "        \"command\": \"$binary_path\""
    echo '      }'
    echo '    }'
    echo '  }'
    echo ""
    success "Installation complete!"
}

main
