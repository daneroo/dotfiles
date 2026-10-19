# AGENTS.md

Daniel's dotfiles and machine setup. `config.yaml` declares what should be
installed; `check.sh` makes the machine match it.

## Layout

- `check.sh`: entry point. Checks brew and go, runs `installDotLinks.sh` (GNU
  Stow links `core/` into `~`), then runs `go/cmd/checkdeps`.
- `config.yaml`: desired state: Homebrew formulae (grouped in sections) and
  casks, asdf versions, npm globals. Lists are sorted by basename, the part after
  the last `/` (`westpoint-io/lazyrsync/lazyrsync` sorts as `lazyrsync`).
- `go/`: `checkdeps`, which reconciles Homebrew, asdf, npm globals, and cached
  shell completions.
- `core/`: Stow package mirrored into `~`. Edit dotfiles here, never the
  symlinked copies in `~`.

## Commands

Commands live in the `Justfile`; run `just` to list them.

- `just ci` is the quality gate (gofmt check, vet, tests). Run it after edits
  and before commits.
- `go test ./go/pkg/asdf -run TestName` runs a single test.
- `just reconcile` runs `./check.sh`, which changes the machine (it installs
  npm globals, for example). Ask before running it.

## Conventions

- Everything here manages a live machine, so every action must be idempotent:
  running it twice is the same as running it once.
- Reconcilers follow one pattern: observe the current state, compare it with
  the desired state from `config.yaml`, then act or print the commands to run.
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
