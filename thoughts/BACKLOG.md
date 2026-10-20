# Backlog

Unscheduled work, grouped by theme. Keep entries brief; move growing detail to
`thoughts/design/` or `thoughts/plans/`. An entry becomes a GitHub issue only
when Daniel asks.

## Next

Leave this section even when empty

## Config

- [ ] cue-config-schema — make `config.cue` the single schema for `config.yaml`:
      add basename sorting, uniqueness, and closed structs; validate from Go via
      `cuelang.org/go` (embedded, so bootstrap needs no `cue` binary); drop the
      hand-written checks in `go/pkg/config`. Reconcile first with the README
      TODO that says "CUE has been declared a disaster area".

## Repo

- [x] rename-master-to-main — rename the default branch from `master` to `main`.
