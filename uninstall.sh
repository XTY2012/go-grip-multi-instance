#!/bin/bash

# go-grip uninstallation script

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

# Check if go-grip is installed
if ! command -v go-grip &> /dev/null; then
    print_error "go-grip is not installed."
    exit 1
fi

# Find where go-grip is installed
GRIP_PATH=$(which go-grip)
print_info "Found go-grip at: $GRIP_PATH"

# Determine if we need sudo
if [[ -w "$(dirname "$GRIP_PATH")" ]]; then
    SUDO=""
else
    SUDO="sudo"
    print_info "Uninstallation requires sudo privileges"
fi

# Confirm uninstallation
read -p "Are you sure you want to uninstall go-grip? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Uninstallation cancelled."
    exit 0
fi

# Remove the binary
print_info "Removing go-grip..."
if $SUDO rm -f "$GRIP_PATH"; then
    print_success "go-grip has been uninstalled successfully!"
else
    print_error "Failed to remove go-grip!"
    exit 1
fi

# Verify uninstallation
if command -v go-grip &> /dev/null; then
    print_error "go-grip is still present in the system!"
    echo "There might be multiple installations."
    exit 1
else
    print_success "Uninstallation verified!"
fi