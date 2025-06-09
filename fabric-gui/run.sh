#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}===== Fabric GUI Runner =====${NC}"

# Check if we should skip pattern loading
if [ "$1" == "--skip-patterns" ]; then
    echo -e "${YELLOW}Skipping pattern loading for faster startup${NC}"
    export FABRIC_GUI_SKIP_PATTERNS=1
fi

# Check if we should enable verbose logging
if [ "$1" == "--verbose" ] || [ "$2" == "--verbose" ]; then
    echo -e "${YELLOW}Enabling verbose logging${NC}"
    export FABRIC_GUI_VERBOSE=1
fi

# Run the application
echo -e "${GREEN}Starting Fabric GUI...${NC}"
go run main.go

# Check exit status
STATUS=$?
if [ $STATUS -ne 0 ]; then
    echo -e "${YELLOW}Fabric GUI exited with status $STATUS${NC}"
else
    echo -e "${GREEN}Fabric GUI exited successfully${NC}"
fi