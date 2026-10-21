# End-to-End Testing

This directory contains end-to-end tests for the dotfiles repository using Docker.

**Note:** _2025-06-03 - 2025-12-09_: my e2e test script corrupted my global git config which caused many repos to commit with the wrong user ("Test User"/test@example.com) see below.

## Overview

The scenario runs `reconfig` in a fresh container from the official
`ghcr.io/homebrew/ubuntu26.04` image, against its own small staged desired
states. `e2e/config.yaml` has a few Linux-compatible formulae, including its
own prerequisites (mise, node for npm, and stow), no casks, bun at `latest`
through mise, and one tiny npm global plus npm itself, which comes with node.
`e2e/config.stray.yaml` adds jq through mise, so going back leaves a stray.
The Ubuntu version matters: current bottles need the glibc of Ubuntu 24.04 or newer;
on older images brew installs its own glibc and gcc first.

Homebrew publishes its images on GitHub's registry (`ghcr.io`); the Docker Hub
copies stopped updating in November 2025. `run.sh` builds with `--pull`, so
each run starts from the newest image.

The container runs on the host's own architecture (arm64 on Apple Silicon), with
no `--platform` flag: under amd64 emulation, tar on Ubuntu 26.04 fails with
`Cannot mkdir: Function not implemented`.

A multi-stage build compiles `reconfig` in a builder stage and copies only the
binary into the clean final stage. The test sets no git config.

## Running the Tests

```bash
just e2e # or ./e2e/run.sh
```

It takes a few minutes, mostly `brew update` and installing node.

## Test Workflow

`e2e/test.sh` asserts output, exit codes, and the machine state at each step.
`reconfig` applies its own plans; the scenario never runs the commands it
prints. It prints how long each `reconfig` run took, each step, and the whole
scenario; `run.sh` prints the total, build included:

0. Setup: remove the committed `mise_completion.bash`, so generating it makes
   a new file, and make `~/.config/bash_includes` a real directory, so Stow
   must link that new file on its own (an unfolded layout).
1. `--help` exits 0; `-n -y` is a usage error, exit 2.
2. `--dry-run` on the fresh machine: exactly the desired formulae are missing,
   mise, npm and stow are not installed, nothing is applied, and no managed
   file, link or completion changes.
3. The default mode with piped input (`y`) announces that input is not
   interactive and previews only: nothing changes.
4. `--yolo` installs the formulae, then uses the new mise, npm and stow in the
   same run: bun through mise, the npm globals, the completions (Bun's written
   once fixed); the container's own `~/.bashrc` makes Stow's plan fail with a
   conflict, exit 1, linking nothing.
5. With the conflicting dotfiles moved aside, `--yolo` links `core/`,
   including the newly generated completion file.
6. A `--yolo` rerun has nothing to apply.
7. A comment and a `[settings]` entry are added to mise's global config;
   `--yolo` with `config.stray.yaml` installs jq through mise.
8. Back to `config.yaml`, with an unwanted formula installed directly: a dry
   run shows the stray, its unwanted version, and the formula to uninstall.
9. `--yolo` removes them, keeping the comment and settings, and a rerun has
   nothing to apply.

Prompts, declines, errors and cancellation are covered with controlled
processes by the Go tests (`just ci`).

## Manual Testing

To start an interactive shell for manual testing:

```bash
docker run --rm -it ghcr.io/homebrew/ubuntu26.04
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
