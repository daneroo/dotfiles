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

**Section**:
A named group of formulae in the desired state, such as `main` or `ai`.
_Avoid_: Category, group

**Basename order**:
The sort rule for package lists: compare the part after the last `/`, so
`westpoint-io/lazyrsync/lazyrsync` sorts as `lazyrsync`.
_Avoid_: Alphabetical order

**Missing tool**:
A package manager or completion tool that isn't installed, so its part of the
desired state is reported and skipped rather than reconciled.
_Avoid_: Missing dependency

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
