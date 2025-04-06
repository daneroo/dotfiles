#!/usr/bin/env bash
set -e

# This is the entrypoint script for the e2e test container
echo "Starting e2e test in Docker container"

# Get script directory for reliable path resolution
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/.." && pwd)"
cd "$REPO_ROOT"

# Run the e2e test script
./e2e/test.sh

# If we get here, the test passed
echo "E2E test completed successfully!"