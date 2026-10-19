# Backlog

Unscheduled work, grouped by theme. Keep entries brief; move growing detail to
`thoughts/design/` or `thoughts/plans/`. An entry becomes a GitHub issue only
when Daniel asks.

## Now

- [ ] asdf-to-mise — replace asdf with mise.

## Config

- [ ] cue-config-schema — make `config.cue` the single schema for `config.yaml`:
      add basename sorting, uniqueness, and closed structs; validate from Go via
      `cuelang.org/go` (embedded, so bootstrap needs no `cue` binary); drop the
      hand-written checks in `go/pkg/config`. Reconcile first with the README
      TODO that says "CUE has been declared a disaster area".
- [ ] remove-deno — promoted to issue #2.

## Closed

- [x] agents-md-rewrite — 2026-10-19: `AGENTS.md` written from the old
      `CLAUDE.md` (Go commands, reconciliation, idempotency kept; Deno and the
      stale Claude Code auth notes dropped); `CLAUDE.md` now imports it.
