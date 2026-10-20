# Executable reconfig: implementation index

The accepted requirements are published in [spec #11](https://github.com/daneroo/dotfiles/issues/11).
The [local spec](../design/reconfig-spec.md) preserves the approved baseline for
repository reviews; GitHub holds current requirements and ticket status.
Implementation has not started.

## Implementation sequence

Use one local branch, `feat/reconfig-execution`, and implement tickets in this
order, with a checked commit for each completed ticket. Do not create parallel
implementation branches or worktrees.

| Ticket | Scope | Blocked by |
| --- | --- | --- |
| [#12](https://github.com/daneroo/dotfiles/issues/12) | Shared runner and npm | None |
| [#13](https://github.com/daneroo/dotfiles/issues/13) | Homebrew upgrades | #12 |
| [#14](https://github.com/daneroo/dotfiles/issues/14) | Homebrew packages | #13 |
| [#15](https://github.com/daneroo/dotfiles/issues/15) | Mise | #12 |
| [#16](https://github.com/daneroo/dotfiles/issues/16) | Completions | #12 |
| [#17](https://github.com/daneroo/dotfiles/issues/17) | Stow | #12 |
| [#18](https://github.com/daneroo/dotfiles/issues/18) | CLI cutover and convergence | #14, #15, #16, #17 |

The sequence is the owner's workflow preference; the native GitHub dependencies
record genuine technical blockers. Fetch full issue bodies and comments before
implementing. Keep the existing CLI working while adding reconciler adapters;
#18 switches the whole program once every managed writer is covered.

Run `just ci` after edits and before commits. Use controlled fixtures and the
container E2E scenario for execution checks. Do not run live reconciliation
without the owner's permission. Keep the owner informed every 30–60 seconds
while working, and report each ticket's checks and completion clearly.

## Retained evidence

- [Claude's original review](../reviews/reconfig-claude-design-review.md) examined
  an earlier draft of this plan. Read it with the
  [assessment and accepted corrections](../reviews/reconfig-design-review-assessment.md).
- [Mise native reconciliation](../research/mise-native-reconciliation.md) records
  exploratory findings, including the staged-config approach later dropped.
- [Mise configuration-only application](../research/mise-config-only-application.md)
  verifies the accepted simpler approach, including literal ordered requests.
- [Homebrew executable removals](../research/homebrew-executable-removals.md)
  checks automatic removal and protection of desired dependencies.
- [Completion and Stow planning](../research/completion-and-stow-planning.md)
  records fresh Bun output and Stow unchanged/needed/conflicting cases.

Research and review documents preserve evidence, including rejected approaches;
they do not override the accepted spec or ticket requirements.

## Document lifecycle

The planning session verified published issues #11–#18 before removing duplicate
discussion notes, the detailed draft plan, and seven local ticket drafts. This
index replaces the draft plan and ticket index. The final local spec, glossary,
research, and reviews are retained as versioned records in the same documentation
commit. Temporary probes and the session handoff remain outside the repository.

The implementation session owns code changes and their documentation updates.
It should not recreate local ticket drafts. Update the existing print-only
reconciler instructions during #18, as required by the spec. Branch creation and
implementation follow this planning cleanup; pushing and PR delivery remain to
be agreed with the owner.
