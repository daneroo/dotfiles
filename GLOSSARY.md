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
Compare the actual state with the desired state, then act or print the commands
that would close the gap.
_Avoid_: Sync, check deps

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
