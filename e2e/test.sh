#!/usr/bin/env bash
set -e

# This script runs tests inside the Docker container
echo "Running e2e tests..."

# Get script directory for reliable path resolution
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/.." && pwd)"
cd "$REPO_ROOT"

# Display container environment info
echo "Container environment:"
uname -a
echo "Homebrew version: $(brew --version | head -1)"
echo "Bash version: ${BASH_VERSION}"

# Set up git config for testing
git config --global user.name "Test User"
git config --global user.email "test@example.com"

# Document the current state before any modifications
echo "Capturing initial state..."
brew list > /tmp/initial_brew_formulae.txt
echo "Initial brew packages: $(wc -l < /tmp/initial_brew_formulae.txt)"

# Test loading configuration
echo "Testing configuration loading..."
checkdeps -c config.yaml -v

echo "All tests passed!"
exit 0