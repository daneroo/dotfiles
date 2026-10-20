# Mise configuration-only application

Date: 2026-10-20. Empirical follow-up to
[native reconciliation](mise-native-reconciliation.md), prompted by the design
review's suggestion to keep native configuration edits separate from explicit
installation operations.

## Scope

The installed Mise executable is `2026.10.15 macos-arm64 (2026-10-19)`.
Experiments used a copy of the existing synthetic `reconfig-fixture` asdf plugin.
All configuration, data, cache, state, system configuration, plugin files, and
working directory were isolated under
`/tmp/reconfig-mise-config-only-dos53f0n/`. `HOME` was unchanged. The fake plugin
creates a shell executable locally and does not download distributions.

Raw command arguments, exit statuses, output, and configuration snapshots are in
the temporary [results.json](/tmp/reconfig-mise-config-only-dos53f0n/results.json).
No live Mise configuration or real tool installations were modified. These are
fixture observations, not guarantees for every backend or native tool option.

## Configuration can be edited without installation

This command succeeded against the isolated global file:

```sh
mise config set --file /tmp/reconfig-mise-config-only-dos53f0n/global.toml \
  --type list tools.reconfig-fixture 1.0,1.1
```

It wrote `reconfig-fixture = ["1.0", "1.1"]`, preserving an existing comment,
`[settings]` with `jobs = 1`, and an `[env]` value. No installation directory was
created. Reversing the argument to `1.1,1.0` preserved that exact order; repeating
the same write preserved the file contents. Mise's installed help and
[configuration-editing documentation](https://mise.jdx.dev/cli/config/set.html)
explicitly distinguish this command from the installation behavior of `use`.

Applied `unuse --global --no-prune reconfig-fixture` removed only its global
request, preserving the comment, settings, environment value, and both installed
fixture versions. Explicit `install` and `uninstall` commands left global
configuration unchanged. These operations can therefore be separate actions
inside one reconciler. The documented configuration-only removal flag is
[`unuse --no-prune`](https://mise.jdx.dev/cli/unuse.html).

This proves string and ordered string-array declarations, not arbitrary native
options objects. `config set --type list` uses comma-separated input. A follow-up
probe found a native way to avoid delimiter ambiguity for distinct strings:

```sh
mise config set --file FILE --type string tools.reconfig-fixture 'a,b'
mise config set --file FILE --append --type string tools.reconfig-fixture '1.0'
```

The first command replaces any existing value with literal string `"a,b"`.
The second converts it to `["a,b", "1.0"]`, preserving the comma inside the
first string. Repeating the complete sequence reproduced the same final
contents; reversing the seed and append values reversed the final array.
Appending an existing value again did not duplicate or reorder it. No installs
occurred, and unrelated configuration survived. These artificial strings test
literal editing, not whether they are valid version requests.

The proposed empty-list reset is unsuitable: `--type list ... ''` writes
`[""]`, not `[]`. Removing that empty string with `--remove --type string`
deletes the tool key when it was the only member. Appending to the absent key
creates an array. Starting directly with the desired first string is simpler.

Consequently, native scalar assignment plus literal appends handles a nonempty
ordered list of distinct opaque strings without parsing their version grammar
or serializing TOML ourselves. It does not preserve repeated identical array
entries or a singleton array's exact shape, and partial application temporarily
leaves an incomplete selection. General native `[tools]` options objects still
need a deliberate editing strategy.

## Native version resolution agrees for tested valid requests

The plugin initially advertised `1.0.0`, `1.0.1`, and `1.1.0`. With no versions
installed, `mise latest reconfig-fixture@SPEC` and
`mise upgrade --global --no-prune --dry-run` against a global request of `SPEC`
produced these results:

| Request | `latest` output | Upgrade preview installation |
| --- | --- | --- |
| `1.0` | `1.0.1` | `1.0.1` |
| `prefix:1.0` | `1.0.1` | `1.0.1` |
| `1.0.0` | `1.0.0` | `1.0.0` |
| `1.0.1` | `1.0.1` | `1.0.1` |
| `9.9` (unadvertised) | Empty stdout, exit 0 | `9.9`, exit 0 |

The last row is a validation limitation: success status alone does not establish
a usable concrete resolution or installability. A planner using `latest` must
handle empty output explicitly. Upgrade preview likewise does not prove a
release exists. Mise documents `latest tool@request` as selecting the newest
available match in its
[version inspection documentation](https://mise.jdx.dev/dev-tools/versions.html).

## Freezing concrete versions avoids a second resolution during apply

An additional sequence simulated a release appearing between planning and apply:

1. Resolve desired requests `1.0` and `1.1`; retain results `1.0.1` and `1.1.0`.
2. Add advertised release `1.0.2` and clear only the isolated fixture cache.
   A fresh `latest ...@1.0` now returns `1.0.2`.
3. Keep the global declarations as `["1.0", "1.1"]` using `config set`.
4. Apply `mise install reconfig-fixture@1.0.1 reconfig-fixture@1.1.0`.

Exactly the approved `1.0.1` and `1.1.0` were installed; `1.0.2` was not.
`ls --installed --json` reported both active with their original global requests.
Repeating the install reported both already installed, with zero new installs.
The configuration did not change. No `upgrade`, `prune`, or combined `use`
operation was needed during application.

This supports a plan containing unchanged request strings plus concrete install
and uninstall targets, all resolved before confirmation. Apply can use
[explicit installation arguments](https://mise.jdx.dev/cli/install.html) rather
than re-resolving floating requests. It preserves target agreement in this
fixture; it does not guarantee download success, transactional rollback,
identical backend artifacts, or protection against concurrent external changes.

The application sequence can edit global requests, install all required concrete
versions, and only then explicitly uninstall extraneous versions. A failure must
stop later actions within the reconciler, especially cleanup. Alternatively,
installing before selecting new globals reduces the interval in which global
requests point at missing installations. Both orders avoid automatic pruning;
the final implementation must choose and document its partial-failure behavior.

## Missing explicit uninstall is already tolerant

`mise uninstall reconfig-fixture@9.9.9` exited 0 with a warning that the version
was not installed. Removing installed `1.0.0` and repeating that command also
exited 0 on repetition, leaving `1.0.1` intact and configuration unchanged.
Thus the tested executable already tolerates a disappeared concrete cleanup
target; no blanket suppression of real uninstall errors is justified.

## Implication for the proposed contract

For the tested declarations, the simpler implementation is viable: delegate
request interpretation to Mise during observation, retain explicit targets in
the approved plan, and use native configuration-only edits plus concrete
install/uninstall commands during apply. Exclusive cleanup remains our explicit
comparison of desired resolutions with installed versions. Native pruning has
different ownership rules and is not substituted for that policy.

General native tool-option objects, backend-specific resolution, lockfiles,
global override/system configurations, and real distribution installation still
need coverage before claiming full native `[tools]` support. These findings
support a design choice; they do not implement it or settle its whole schema.
