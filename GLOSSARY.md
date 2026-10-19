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
Compare the actual state with the desired state, then print the commands that
would close the gap: installs and removals alike. Reconciling never changes the
machine itself.
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

**Versioned tool**:
A program the machine keeps at specific versions, possibly several side by
side, one of them the default. Python and Bun are versioned tools.
_Avoid_: Runtime, plugin, SDK

**Version spec**:
How the desired state names a versioned tool's version: a prefix such as
`3.12` (the newest matching release) or `latest`.
_Avoid_: Pin, constraint
