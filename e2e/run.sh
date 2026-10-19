#!/usr/bin/env bash
set -e

ECHO_PREFIX="[E2E Test]"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/.." && pwd)"

cd "$REPO_ROOT"

echo "$ECHO_PREFIX Building Docker image..."
# --pull: check for a newer base image, so :latest never goes stale
docker build --pull -t dotfiles-e2e -f "$DIR/Dockerfile" .

echo "$ECHO_PREFIX Running e2e tests in Docker container..."
docker run --rm dotfiles-e2e

echo "$ECHO_PREFIX E2E tests completed successfully in $((SECONDS / 60))m$((SECONDS % 60))s (build included)!"