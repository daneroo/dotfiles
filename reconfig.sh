#!/usr/bin/env bash

echo "# Configuration and Dependencies"

DOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DOTDIR" || exit 1

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
    echo " install it with:"
    echo " brew install go"
    exit 1
else
    echo "✓ - brew is installed and go is available"
fi

echo
echo "# Dependencies"
# Built rather than go run, which reports every failure as exit status 1:
# reconfig exits 2 for a usage error and 130 when interrupted.
BIN_DIR="$(mktemp -d)" || exit 1
trap 'rm -rf "$BIN_DIR"' EXIT
go build -o "$BIN_DIR/reconfig" ./go/cmd/reconfig || exit 1
"$BIN_DIR/reconfig" "$@"
code=$?
if [ $code -ne 0 ]; then
    echo "Exiting: reconfig failed with error code: ${code}"
fi
exit "${code}"
