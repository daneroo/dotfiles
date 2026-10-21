# AGENTS.md

Daniel's dotfiles and machine setup. `config.yaml` declares what should be
installed; `reconfig.sh` makes the machine match it.

## Layout

- `reconfig.sh`: entry point. Checks brew and go, then builds and runs
  `go/cmd/reconfig` from the repository root, forwarding its arguments
  (`./reconfig.sh --dry-run`). `check.sh` is retired: it only says to run
  `reconfig.sh`.
- `config.yaml`: desired state: Homebrew formulae (grouped in sections) and
  casks, mise versioned tools, npm globals. Lists are sorted by basename, the
  part after the last `/` (`westpoint-io/lazyrsync/lazyrsync` sorts as
  `lazyrsync`); mise version specs are not sorted, the first is the default.
- `go/`: `reconfig`, which reconciles, in order: Homebrew upgrades, Homebrew
  packages, mise versioned tools, npm globals, shell completions, and Stow
  links of `core/` into `~`. `go/pkg/run` is the runner; `go/pkg/proc` runs
  native programs.
- `core/`: Stow package mirrored into `~`. Edit dotfiles here, never the
  symlinked copies in `~`.

## Commands

Commands live in the `Justfile`; run `just` to list them.

- `just ci` is the quality gate (gofmt check, vet, tests). Run it after edits
  and before commits.
- `go test ./go/pkg/config -run TestName` runs a single test.
- `just e2e` runs the end-to-end scenario in a fresh Homebrew container against
  its own staged desired states, `e2e/config*.yaml` (needs Docker, takes
  minutes).
- `just reconcile` runs `./reconfig.sh` (options pass through), which asks
  before applying each plan and changes the machine when approved. Ask before
  running it, in any mode: even `--dry-run` refreshes Homebrew's metadata.
  Validate with tests and the e2e scenario, never by reconciling this machine.

## Conventions

- Everything here manages a live machine, so every action must be idempotent:
  running it twice is the same as running it once.
- Reconcilers follow one pattern (`run.Reconciler`): observe the actual state,
  compare it with the desired state from `config.yaml`, and return a plan: a
  report to show and the actions to apply. They never prompt and never choose
  whether to apply; the runner does, by execution mode:
  - default: ask before applying each actionable plan; when input is not a
    terminal, preview only, as `--dry-run` does
  - `--dry-run` (`-n`): show every plan, apply nothing
  - `--yolo` (`-y`): apply every actionable plan without asking

  A decline or a real error previews the rest of the run. Plans keep the
  arguments they run; displayed text is never run.
- Reconcilers take a `proc.Exec`, so tests substitute controlled processes.
- A reconciler checks for its tool first; a missing tool makes its plan
  unavailable: reported and skipped, never an error. Only real errors exit
  non-zero (1; 2 for a usage error; 130 when interrupted).
- Go: standard style and `gofmt`; callers before helpers (top-down); explicit
  error returns; document exported identifiers; add tests for new behavior.
- Markdown: follow Markdownlint rules.

## Agent skills

### Issue tracker

GitHub Issues via `gh`; pre-issue backlog and working notes in `thoughts/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
