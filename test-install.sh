#!/bin/bash

# Test installation script

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Testing go-grip installation...${NC}"

# Run install script
echo "Running install.sh..."
./install.sh

# Test if go-grip is available
echo -e "\n${YELLOW}Checking installation...${NC}"
which go-grip

# Show version/help
echo -e "\n${YELLOW}Testing go-grip command...${NC}"
go-grip --help

echo -e "\n${GREEN}✅ Installation test completed!${NC}"
echo "You can now use 'go-grip' from anywhere in your system."