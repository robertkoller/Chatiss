#!/usr/bin/env bash
# Local script for running or stopping the Chatiss client.
# Usage:
#   ./scripts/install.sh       — build and run the client
#   ./scripts/install.sh -s    — kill all running client processes

set -e

if [ "${1:-}" = "-s" ]; then
    pkill -f chatiss-client 2>/dev/null && echo "All client processes killed." || echo "No client processes were running."
    exit 0
fi

cd "$(dirname "$0")/.."
mkdir -p bin
go build -o bin/chatiss-client ./cmd/client/
./bin/chatiss-client
