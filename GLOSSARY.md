# Machine Configuration

Keeping a machine's installed software and dotfiles in line with what this repo
declares.

## Language

**Desired state**:
The software a machine should have, as declared in `config.yaml`.
_Avoid_: Spec, manifest

**Actual state**:
The software installed on the machine right now.
_Avoid_: Current config

**Reconcile**:
Observe the actual state, compare it with the desired state, and compute actions
to close the gap. Execution of those actions follows the selected execution mode.
_Avoid_: Sync, check deps

**Reconciler**:
The part of reconfig responsible for bringing a particular kind of actual state
into agreement with its desired state, such as dotfile links or versioned tools.
_Avoid_: Step, execution section, area

**Plan**:
The changes a reconciler proposes to bring its observed actual state toward the
desired state.
_Avoid_: Command dump

**Action**:
A change within a reconciler's plan, such as selecting a global tool version and
installing what it needs.
_Avoid_: Item, shell line

**Report**:
What a plan shows before anything is applied: observations, then its actions
grouped by kind. It is only displayed; the plan keeps the arguments it runs.
_Avoid_: Output, printed commands

**Unavailable**:
A plan that could not be made, because of a missing tool or an unmet
prerequisite, such as pending Homebrew upgrades before package changes. It is
reported and skipped; it is not an error.
_Avoid_: Failed, blocked

**Execution mode**:
Whether the runner applies actionable plans: confirm (the default; ask before
each one), dry run (apply none), or YOLO (apply all without asking). Without a
terminal, confirm previews only, as a dry run does. A decline or a real error
turns the rest of the run into a dry run.
_Avoid_: Force, check mode

**Section**:
A named group of formulae in the desired state, such as `main` or `ai`.
_Avoid_: Category, group

**Basename order**:
The sort rule for package lists: compare the part after the last `/`, so
`westpoint-io/lazyrsync/lazyrsync` sorts as `lazyrsync`.
_Avoid_: Alphabetical order

**Missing tool**:
A package manager or completion tool that isn't installed, so its part of the
desired state is reported and skipped rather than reconciled: its plan, or that
part of it, is unavailable.
_Avoid_: Missing dependency

**Completion script**:
The Bash code that completes a tool's commands and arguments on Tab. Reconfig
produces one for each tool it manages completions for.
_Avoid_: Completion file, cached completion

**On-demand loading**:
Bash reads a tool's completion script the first time that tool is completed,
not when the shell starts.
_Avoid_: Lazy loading, autoload

**Shadowed**:
A completion script that bash never reads, because another script for the same
tool was already loaded when the shell started. Completion still works.
_Avoid_: Overridden, duplicate

**Versioned tool**:
A program the machine keeps at specific versions, possibly several side by
side, one of them the default. Python and Bun are versioned tools.
_Avoid_: Runtime, plugin, SDK

**Stray**:
A tool listed in mise's global configuration that the desired state doesn't
list, typically one removed from `config.yaml`.
_Avoid_: Orphan, leftover

**Version spec**:
How the desired state names a versioned tool's version: a prefix such as
`3.12` (the newest matching release), `latest`, or `lts` (the newest
long-term support release, for tools that have one, such as node).
_Avoid_: Pin, constraint
