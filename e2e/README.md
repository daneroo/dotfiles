# End-to-End Testing

This directory contains end-to-end tests for the dotfiles repository using Docker.

**Note:** _2025-06-03 - 2025-12-09_: my e2e test script corrupted my global git config which caused many repos to commit with the wrong user ("Test User"/test@example.com) see below.

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

## Bad Git user

_2025-06-03 - 2025-12-09_: my e2e test script corrupted my global git config which caused many repos to commit with the wrong user ("Test User"/test@example.com)

Check that it is fixed: `git config --global --list`.

```bash
# find all the commit by Test User
find /Users/daniel/Code -name ".git" -type d 2>/dev/null | while read gitdir; do
  repo=$(dirname "$gitdir")
  count=$(git -C "$repo" log --author="Test User" --oneline 2>/dev/null | wc -l | tr -d ' ')
  if [ "$count" -gt 0 ]; then
    echo "$count commits: $repo"
  fi
done | sort -rn
```

Found 7 affected repos with a total of 171 commits by "Test User":

| Commits | Repository                                 |
| ------- | ------------------------------------------ |
| 94      | ~/Code/iMetrical/nix-garden                |
| 44      | ~/Code/iMetrical/nx-audiobook              |
| 13      | ~/Code/iMetrical/ai-garden                 |
| 9       | ~/Code/iMetrical/wifidan                   |
| 6       | ~/Code/iMetrical/scrobbleCast              |
| 3       | ~/Code/iMetrical/chromebook-asus-flip-C436 |
| 2       | ~/Code/iMetrical/im-qcic                   |

Also, don't forget this repo (`~/.dotfiles`) itself.

```bash
git -C ~/.dotfiles log --author="Test User" --oneline
```
