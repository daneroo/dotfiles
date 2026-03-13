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
# reusing real values, becaus of earlier global git pollution in failed test
# These values are isolated, but they escaped in the past
git config --global user.name "Daniel Lauzon"
git config --global user.email "daniel.lauzon@gmail.com"

# Document the current state before any modifications
echo "Capturing initial state..."
brew list > /tmp/initial_brew_formulae.txt
echo "Initial brew packages: $(wc -l < /tmp/initial_brew_formulae.txt)"

# Test loading configuration
echo "Testing configuration loading..."
checkdeps -c config.yaml -v

echo "All tests passed!"
exit 0