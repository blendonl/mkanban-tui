#!/bin/bash
# Installation script for mkanban-tui

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default installation prefix
PREFIX="${PREFIX:-/usr/local}"
BINDIR="${PREFIX}/bin"
SYSTEMD_USER_DIR="${HOME}/.config/systemd/user"
COMPLETIONS_DIR="${PREFIX}/share"

# Print functions
print_info() {
    echo -e "${GREEN}==>${NC} $1"
}

print_error() {
    echo -e "${RED}Error:${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

# Check if running as root for system-wide install
if [ "$PREFIX" = "/usr" ] || [ "$PREFIX" = "/usr/local" ]; then
    if [ "$EUID" -ne 0 ]; then
        print_error "System-wide installation requires root privileges."
        echo "Please run with sudo or set PREFIX to a user directory:"
        echo "  sudo ./install.sh"
        echo "  PREFIX=\$HOME/.local ./install.sh"
        exit 1
    fi
fi

# Check dependencies
print_info "Checking dependencies..."
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.24 or later."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_info "Found Go version: $GO_VERSION"

# Build binaries
print_info "Building mkanban TUI client..."
go build -trimpath -ldflags "-s -w" -o mkanban ./cmd/mkanban

print_info "Building mkanbad daemon..."
go build -trimpath -ldflags "-s -w" -o mkanbad ./cmd/mkanbad

# Install binaries
print_info "Installing binaries to $BINDIR..."
mkdir -p "$BINDIR"
install -m755 mkanban "$BINDIR/mkanban"
install -m755 mkanbad "$BINDIR/mkanbad"

# Generate and install shell completions
print_info "Generating shell completions..."
./mkanban completion bash > mkanban.bash
./mkanban completion zsh > mkanban.zsh
./mkanban completion fish > mkanban.fish

if [ "$EUID" -eq 0 ]; then
    # System-wide completions
    print_info "Installing shell completions (system-wide)..."
    mkdir -p "${PREFIX}/share/bash-completion/completions"
    mkdir -p "${PREFIX}/share/zsh/site-functions"
    mkdir -p "${PREFIX}/share/fish/vendor_completions.d"

    install -m644 mkanban.bash "${PREFIX}/share/bash-completion/completions/mkanban"
    install -m644 mkanban.zsh "${PREFIX}/share/zsh/site-functions/_mkanban"
    install -m644 mkanban.fish "${PREFIX}/share/fish/vendor_completions.d/mkanban.fish"
else
    # User completions
    print_info "Installing shell completions (user)..."
    mkdir -p "${HOME}/.local/share/bash-completion/completions"
    mkdir -p "${HOME}/.local/share/zsh/site-functions"
    mkdir -p "${HOME}/.config/fish/completions"

    install -m644 mkanban.bash "${HOME}/.local/share/bash-completion/completions/mkanban"
    install -m644 mkanban.zsh "${HOME}/.local/share/zsh/site-functions/_mkanban"
    install -m644 mkanban.fish "${HOME}/.config/fish/completions/mkanban.fish"
fi

# Install systemd user service
print_info "Installing systemd user service..."
mkdir -p "$SYSTEMD_USER_DIR"
install -m644 systemd/mkanbad.service "$SYSTEMD_USER_DIR/mkanbad.service"

# Update systemd user service to use correct binary path
sed -i "s|/usr/bin/mkanbad|$BINDIR/mkanbad|g" "$SYSTEMD_USER_DIR/mkanbad.service"

# Clean up build artifacts
print_info "Cleaning up..."
rm -f mkanban.bash mkanban.zsh mkanban.fish

print_info "Installation complete!"
echo ""
echo "To enable the daemon to start automatically:"
echo "  systemctl --user daemon-reload"
echo "  systemctl --user enable --now mkanbad.service"
echo ""
echo "To use mkanban:"
echo "  mkanban                    # Launch TUI"
echo "  mkanban board list         # List boards via CLI"
echo "  mkanban task create ...    # Create tasks via CLI"
echo ""
echo "For help:"
echo "  mkanban --help"
echo ""

if [ "$BINDIR" != "/usr/bin" ] && [ "$BINDIR" != "/usr/local/bin" ]; then
    print_warning "Binaries installed to $BINDIR"
    echo "Make sure $BINDIR is in your PATH:"
    echo "  export PATH=\"$BINDIR:\$PATH\""
    echo ""
fi
