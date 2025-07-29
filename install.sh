#!/bin/bash

# go-grip installation script
# This script builds and installs go-grip globally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go first."
    echo "Visit https://golang.org/dl/ for installation instructions."
    exit 1
fi

print_info "Go version: $(go version)"

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Build the binary
print_info "Building go-grip..."
if make build; then
    print_success "Build successful!"
else
    print_error "Build failed!"
    exit 1
fi

# Determine installation directory
if [[ "$OSTYPE" == "darwin"* ]] || [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # macOS or Linux
    INSTALL_DIR="/usr/local/bin"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    # Windows (Git Bash/Cygwin)
    INSTALL_DIR="/usr/bin"
else
    print_error "Unsupported operating system: $OSTYPE"
    exit 1
fi

# Check if we have write permission to install directory
if [[ -w "$INSTALL_DIR" ]]; then
    SUDO=""
else
    SUDO="sudo"
    print_info "Installation requires sudo privileges"
fi

# Install the binary
print_info "Installing go-grip to $INSTALL_DIR..."
if $SUDO cp bin/go-grip "$INSTALL_DIR/"; then
    print_success "go-grip installed successfully!"
else
    print_error "Installation failed!"
    exit 1
fi

# Make sure it's executable
$SUDO chmod +x "$INSTALL_DIR/go-grip"

# Verify installation
if command -v go-grip &> /dev/null; then
    print_success "Installation verified!"
    echo ""
    print_info "go-grip has been installed successfully!"
    echo "Version: $(go-grip --version 2>&1 | head -n1 || echo 'version info not available')"
    echo ""
    echo "Usage examples:"
    echo "  go-grip                    # Serve current directory"
    echo "  go-grip README.md          # Serve a specific file"
    echo "  go-grip docs/              # Serve a documentation directory"
    echo "  go-grip -p 8080 docs/      # Use custom port"
    echo ""
    echo "Features:"
    echo "  - Multi-file markdown serving with directory support"
    echo "  - Wiki-style links: [[Page Name]] → /page-name.md"
    echo "  - Auto-reload on file changes"
    echo "  - GitHub-style markdown rendering"
    echo ""
else
    print_error "Installation verification failed!"
    echo "Please check if $INSTALL_DIR is in your PATH"
    exit 1
fi