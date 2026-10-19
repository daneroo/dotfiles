# End-to-End Testing

This directory contains end-to-end tests for the dotfiles repository using Docker.

**Note:** _2025-06-03 - 2025-12-09_: my e2e test script corrupted my global git config which caused many repos to commit with the wrong user ("Test User"/test@example.com) see below.

## Overview

The scenario runs `reconfig` in a fresh container from the official
`homebrew/ubuntu24.04` image, against its own small desired state,
`e2e/config.yaml` (a few Linux-compatible formulae, no casks, no asdf, one tiny
npm global). Ubuntu 24.04 matters: its glibc 2.39 is what current bottles need;
on the older `homebrew/brew` image (glibc 2.35) brew installs its own glibc and
gcc first.

A multi-stage build compiles `reconfig` in a builder stage and copies only the
binary into the clean final stage. The test sets no git config.

## Running the Tests

```bash
just e2e # or ./e2e/run.sh
```

It takes a few minutes, mostly `brew update` and installing node.

## Test Workflow

`e2e/test.sh` asserts output, exit codes, and installed packages at each step:

1. First run reports exactly the desired formulae as missing and exits 0
   (asdf and npm are reported as not installed, skipping).
2. Run the `brew install` line `reconfig` printed.
3. Rerun: nothing missing, nothing extraneous.
4. Install one formula not in the desired state; rerun: it is reported
   extraneous.
5. Install node; rerun: `reconfig` installs the npm global; rerun again:
   nothing to do.

## Manual Testing

To start an interactive shell for manual testing:

```bash
docker run --rm --platform linux/amd64 -it homebrew/ubuntu24.04
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
