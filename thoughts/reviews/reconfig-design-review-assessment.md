# Assessment of Claude's design review

Status: both implementation revisions were accepted and incorporated in
[issue #11](https://github.com/daneroo/dotfiles/issues/11). No implementation has
started.

The original draft plan was replaced by the short
[implementation index](../plans/reconfig-executable-plans.md) after publication.
The raw review below is historical; the final spec and tickets define the work.

## Review record

The [full Claude response](reconfig-claude-design-review.md) is preserved without
edits. The installed Claude Code CLI was version 2.1.304; the returned model
identifier was `claude-opus-5-5`. The run completed successfully in 4 minutes
49 seconds, with no permission denials and no delegated agents.

The review used the following command from the repository root. The input
specified the accepted requirements, requested independent codebase-design
criticism, and directed Claude to read the draft, research, skills, actual code,
and tests. Only reading and searching tools were available; shell and editing
tools and MCP were unavailable.

```sh
claude --print --safe-mode --restricted --permission-mode plan \
  --tools Read,Glob,Grep --allowedTools Read,Glob,Grep \
  --strict-mcp-config --no-session-persistence --output-format json \
  < /tmp/reconfig-claude-review-prompt.txt
```

The JSON result is not streamed, so it provided no progress visibility while
running. Use streaming output for a future external review. The report is review
input rather than authority over the user's accepted requirements.

## Initial assessment

Retain the shared runner and the separation between reporting and execution.
The reviewed interface fits the distinct implementations and keeps confirmation
and mode transitions in one place.

The process-exit, cancellation, terminal-I/O, and Stow-no-op findings are concrete
implementation obligations, now added to the draft rather than introducing more
public abstractions. Real planning errors follow the same stop-applying rule as
execution errors; missing tools and unmet prerequisites remain non-errors.

The staged Mise design merits simplification. The current code already asks
Mise to resolve version requests and has a tested comparison function. Native
configuration-only edits followed by exact reported installations may retain
those strengths without temporary TOML composition or a new parsing dependency.
The [isolated config-only probes](../research/mise-config-only-application.md)
verified request ordering, preservation of unrelated configuration, and exact
installations despite a newer release appearing after observation. This
supports the simpler path. Install before changing global selections to reduce
the chance of selecting a missing version after an installation failure.

Claude's F3 claim that removing roots necessarily leaves dependencies behind is
incomplete for current Homebrew: uninstall automatically runs autoremove unless
disabled. That can also remove a desired formula originally installed as a
dependency, so it is not a faithful desired-state cleanup policy. Native source
supports removing a dependent and dependency in the same removal set. Prefer an
explicit complete removal set with native autoremove disabled, retaining native
dependency guards. Dependencies of missing desired packages and cask-to-cask
requirements must also be protected before proposing removals. These are
correctness requirements made material by enabling execution, not presentation
changes.

The Bun finding correctly identified a contradiction in the draft's account of
the existing code. [Isolated installed-tool probes](../research/completion-and-stow-planning.md)
confirmed fresh stdout generation with no fixture changes, and the current code
discards that stdout before reading a potentially stale file. The same research
verified Stow warning-only unchanged output and operation/conflict cases. The
plan now records those concrete implementation obligations.

## Accepted revisions

1. Use the existing native Mise resolution/comparison approach, plus native
   configuration-only editing and explicit reported installations/removals.
   Preserve opaque version request strings and support scalar/ordered-list
   declarations; defer arbitrary options objects until supported faithfully.
   Drop staged TOML composition for this first implementation.
2. Make Homebrew's removal plan explicit and complete, suppress automatic
   autoremove during managed mutations, and protect all desired dependencies.
   Do not promise one-run convergence while preserving roots-only execution.

The shared runner, three modes, per-reconciler confirmation, existing output
style, and decline/failure transitions remain unchanged. The standard Go flag
parser is retained with custom conventional help; no CLI dependency is added.
