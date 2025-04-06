# End-to-End Testing

This directory contains end-to-end tests for the dotfiles repository using Docker.

## Overview

The E2E tests run in a clean Docker container based on the vanilla `homebrew/brew` image.
This ensures that the tests run in an environment similar to a fresh installation.

## Implementation Details

The test setup uses a multi-stage Docker build to:

- Build the Go binary in an isolated builder stage
- Copy only the compiled binary to the final clean environment
- Run tests against a pristine Homebrew installation

This approach ensures we don't pollute the test environment with build dependencies.

## Running the Tests

To run the E2E tests:

```bash
./e2e/run.sh
```

This script will:

- Build a Docker image with the Dockerfile in this directory
- Run the container with the current dotfiles repository mounted
- Execute the test script inside the container

## Test Workflow

The testing process:

- Captures the initial state of the container (pre-installed packages)
- Loads and verifies the configuration
- Verifies that our tools can correctly identify missing packages

## Manual Testing

To start an interactive shell for manual testing:

```bash
docker run --rm --platform linux/amd64 -it homebrew/brew
```
