#!/usr/bin/env bash
set -e

ECHO_PREFIX="[E2E Test]"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/.." && pwd)"

cd "$REPO_ROOT"

echo "$ECHO_PREFIX Building Docker image..."
docker build --platform linux/amd64 -t dotfiles-e2e -f "$DIR/Dockerfile" .

echo "$ECHO_PREFIX Running e2e tests in Docker container..."
docker run --rm --platform linux/amd64 -t dotfiles-e2e

echo "$ECHO_PREFIX E2E tests completed successfully!"