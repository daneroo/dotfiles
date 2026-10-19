#!/usr/bin/env bash
# Entrypoint of the e2e container: runs the scenario.
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/test.sh"
