#!/usr/bin/env bash
# Kept for muscle memory: reconfig.sh is the real entry point.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/reconfig.sh" "$@"
