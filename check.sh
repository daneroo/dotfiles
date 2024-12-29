#!/usr/bin/env bash

echo "# Configuration and Dependencies"

DOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo
echo "## Bootstrap"
# Check for brew
if ! command -v brew &>/dev/null; then
    echo "✗ - brew is missing, install it with:"
    echo '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
    exit 1
fi
# Check for go
if ! command -v go &>/dev/null; then
    echo "✓ - brew is installed but go is missing"
    echo " temporarily install it with (it may be replaced with an asdf version later):"
    echo " brew install go"
    exit 1
else
    echo "✓ - brew is installed and go is available"
fi

echo
./installDotLinks.sh

echo
echo "# Dependencies"
go run ./go/cmd/checkdeps/main.go
code=$?
if [ $code -ne 0 ]; then
    echo "Exiting: checkdeps failed with error code: ${code}"
    exit "${code}"
fi

