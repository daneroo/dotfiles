# Backlog

Unscheduled work, grouped by theme. Keep entries brief; move growing detail to
`thoughts/design/` or `thoughts/plans/`. An entry becomes a GitHub issue only
when Daniel asks.

## Now

- [ ] asdf-to-mise — promoted to issue #5.
- [ ] claude-code-via-mise — consider managing Claude Code (and other npm-style
      tools) as versioned tools through mise.

## Config

- [ ] cue-config-schema — make `config.cue` the single schema for `config.yaml`:
      add basename sorting, uniqueness, and closed structs; validate from Go via
      `cuelang.org/go` (embedded, so bootstrap needs no `cue` binary); drop the
      hand-written checks in `go/pkg/config`. Reconcile first with the README
      TODO that says "CUE has been declared a disaster area".

## Repo

- [ ] rename-master-to-main — rename the default branch from `master` to `main`.
