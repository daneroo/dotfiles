# Mise native reconciliation

Date: 2026-10-20. Findings for the reconfig output and execution discussion;
these findings do not settle the design.

The final [spec #11](https://github.com/daneroo/dotfiles/issues/11) drops the
staged-global-config approach explored below. The accepted simpler approach is
verified in [configuration-only application](mise-config-only-application.md).

## Scope and evidence

The installed executable reports `2026.10.15 macos-arm64 (2026-10-19)`.
Installed CLI help was inspected, and an isolated local experiment exercised
the native commands. Official documentation and upstream source explain the
documented behavior; observations below are labeled empirical because they
describe this executable and fixture, rather than every backend or version.

The experiment used a synthetic local asdf plugin named `reconfig-fixture`,
advertising only `1.0.0`, `1.0.1`, and `1.1.0`. Installation created a small local
fixture, without fetching a real language distribution. Each scenario used
isolated configuration, data, cache, state, and system directories under
`/tmp/reconfig-mise-probe-fbhq2f7s/`. No live global configuration or real tool
installation was changed. These were not tests of the Python backend, Python
virtual environments, downloads, compilation, or actual project configurations.

The original empirical record is
[`results.json`](/tmp/reconfig-mise-probe-fbhq2f7s/results.json), a temporary
artifact recording command arguments, exit codes, output, and before/after
configuration. Relevant results are retained below so the findings survive
cleanup of that artifact. Only the scenarios explicitly labeled as applied
changed fixture files; previews left their configuration and installations
unchanged.

## What the native commands cover

- `mise use --global` installs missing versions and records requests in a global
  configuration. `--fuzzy` keeps the requested version spec rather than pinning
  the concrete result. `--remove` removes named tools from the configuration.
  The source replaces the named tool's version list and then saves; its removal
  path does not invoke pruning. It does not automatically remove unnamed tools
  from desired state. [Use documentation](https://mise.jdx.dev/cli/use.html),
  [use implementation](https://github.com/jdx/mise/blob/main/src/cli/use.rs)
- `mise install` installs requested versions without adding tools to
  configuration. Without arguments it uses the loaded configuration; explicit
  `tool@spec` arguments select requests directly.
  [Install documentation](https://mise.jdx.dev/cli/install.html)
- `mise upgrade --global` operates on global and system requests, excluding
  project configuration, project lockfiles, and project environment settings.
  Without `--bump` it retains version ranges. Old upgraded versions are normally
  scheduled for cleanup after the configured grace period; `--prune` requests
  immediate cleanup, while `--no-prune` retains them without scheduling cleanup.
  [Upgrade documentation](https://mise.jdx.dev/cli/upgrade.html),
  [global scope release note](https://github.com/jdx/mise/blob/main/release-notes/2026.10.7.md),
  [deferred cleanup release note](https://github.com/jdx/mise/blob/main/release-notes/2026.8.15.md)
- `mise unuse --global` removes global requests and prunes installations no
  remaining tracked configuration or tool stub needs. `--no-prune` keeps the
  installations. This differs from `use --remove`.
  [Unuse documentation](https://mise.jdx.dev/cli/unuse.html)
- `mise prune --tools` removes installations unused by tracked configurations
  and tool stubs, while retaining installations used by running processes. A
  tool-name argument narrows the tools considered; this is not historical
  ownership by the global configuration.
  [Prune documentation](https://mise.jdx.dev/cli/prune.html)

Installed help and help-only argument parsing additionally confirmed:

| Interface | Installed executable |
| --- | --- |
| `use --dry-run`, `use --dry-run-code` | Supported |
| `use --no-install` | Rejected as an unexpected argument |
| `install --dry-run`, `install --dry-run-code` | Supported |
| `install --global` | Rejected as an unexpected argument |
| `upgrade --global --dry-run` | Supported |
| `unuse --global --no-prune` | Supported |
| `unuse --dry-run` | Rejected as an unexpected argument |
| `prune --tools --dry-run` | Supported |

Consequently, a proposed command must be checked against the installed
interface. In particular, config-only `use --no-install` is unavailable here.
Installed `mise config set --help` explicitly describes config editing without
installing tools, including `--global` and typed list values.
[Config set documentation](https://mise.jdx.dev/cli/config/set.html)

## Empirical scenarios

In this table, `fixture` abbreviates `reconfig-fixture`; the commands in the
evidence use its full name. All commands ran against isolated directories.

| Starting actual state and request | Native operation | Observed result |
| --- | --- | --- |
| No configuration or installation | `use -g --fuzzy --dry-run fixture@1.0` | Would install `1.0.1` and record the `1.0` request. |
| Global `1.0`; only `1.0.0` installed | `use -g --fuzzy --dry-run fixture@1.0` | Reused `1.0.0`; did not propose installing `1.0.1`. |
| Same older matching installation | `install --dry-run fixture@1.0` | Would install `1.0.1`; configuration unchanged. |
| Same older matching installation | `upgrade --global --dry-run` | Would install `1.0.1` and schedule `1.0.0` for pruning after 24 hours. |
| Global `1.0`; `1.0.1` already installed | `use -g --fuzzy --dry-run-code fixture@1.0` | Reported an update and exited **1**, despite unchanged request and installation. |
| Same already converged state | `upgrade --global --dry-run-code` | Exited **0**; warned about `1.1.0` outside the configured range. |
| Global `1.0`; desired request changes to `1.1` | `use -g --fuzzy --dry-run fixture@1.1` | Would install `1.1.0` and update the request; preview left the old config unchanged. |
| Global `1.0`; no installation | `upgrade --global --dry-run` | Would install `1.0.1`; missing requests are included. |
| Global `["1.0", "1.1"]`; only `1.0.0` installed | `use -g --fuzzy --dry-run fixture@1.0 fixture@1.1` | Reused `1.0.0` and would install `1.1.0`. |
| Same multiple-request state | `upgrade --global --dry-run` | Would install both `1.0.1` and `1.1.0`, and schedule `1.0.0` for pruning after 24 hours. |

These results distinguish selecting a usable installed version with `use` from
seeking the newest matching version with the tested `install` and `upgrade`
invocations. `use` alone is insufficient to enforce this repository's existing
meaning of a version spec as the newest matching release.

They also establish that `use --dry-run-code` is not a reliable equality test
between the desired request and actual state: it reported work in an already
converged fixture. This is consistent with the inspected source, which renders
a dry-run addition when supplied versions are present, rather than comparing
serialized configuration for equality.
[Use dry-run implementation](https://github.com/jdx/mise/blob/main/src/cli/use.rs)

### Applied fixture changes and ordered requests

An applied `use -g --fuzzy fixture@1.0` installed synthetic `1.0.1` and wrote:

```toml
[tools]
reconfig-fixture = "1.0"
```

Applied multi-version `use` retained `["1.0", "1.1"]`. Repeating it in reverse
argument order changed the configuration to `["1.1", "1.0"]`, without installing
another version. This confirms preservation of the desired request order in
this fixture. Mise documents that the first version supplies the unversioned
command. [Version documentation](https://mise.jdx.dev/dev-tools/versions.html)

### Removal is separate from upgrading

Applied `use -g --remove fixture` emptied the fixture's global configuration.
Subsequent `ls --installed --json` still reported `1.0.0`, now inactive.
`upgrade --global --dry-run` reported everything up to date and did not propose
removing it. `prune --tools --dry-run` did propose removing that installation
because no tracked configuration or tool stub required it.

This is empirical evidence that upgrading configured requests is not complete
cleanup of installations after requests are removed. The native commands have
separate responsibilities; a complete reconfig plan still needs to account for
strays and installation cleanup.

### A staged global configuration can preview desired state

The experiment selected a temporary desired global configuration containing
request `1.1` and ran `upgrade --global --dry-run` against the same isolated fixture
installation state. It proposed installing `1.1.0`. Reading the original sandbox
global configuration afterward still returned `1.0`.

This demonstrates a way to ask Mise about desired requests before replacing
the actual global configuration. It does not establish a complete transaction,
a preview of global config edits, or cleanup of installations outside those
requests. It also does not test arbitrary native settings or the Python backend.

### Outdated JSON loses a multi-version entry

With requests `["1.0", "1.1"]` and only `1.0.0` installed,
`outdated --global` displayed two rows:

```text
requested  current    latest
1.0        1.0.0      1.0.1
1.1        [MISSING]  1.1.0
```

`outdated --global --json` returned just one object keyed by
`reconfig-fixture`, containing request `1.1`, `current: null`, and latest
`1.1.0`. The `1.0` upgrade was absent. The upgrade dry-run still displayed both
installations and the old-version cleanup.

Upstream JSON rendering inserts each result into a map keyed by tool name,
which explains the observed overwrite. Do not treat this JSON as a complete
multi-version plan for the tested executable.
[Outdated JSON implementation](https://github.com/jdx/mise/blob/main/src/cli/outdated.rs)

## Implications for the discussion

### Native validation through staged planning

Additional isolated probes used `upgrade --global --dry-run` with temporary
global configurations and the same synthetic plugin. A native request
`prefix:1.0` was accepted and resolved to `1.0.1`. Malformed TOML and an integer
tool declaration were both rejected with exit status 1 and configuration parse
errors. This supports delegating parsing and native declaration validation to
Mise while planning, rather than reproducing its grammar in reconfig.

This is not a guarantee that a release can be installed: request `9.9`, absent
from the fixture's advertised release list, still produced `Would install
reconfig-fixture@9.9` and exited 0. Backend-specific checks can occur later.
The temporary empirical record is
[`validation-results.json`](/tmp/reconfig-mise-probe-fbhq2f7s/validation-results.json).
The native configuration shapes are documented in
[Mise's schema](https://github.com/jdx/mise/blob/main/schema/mise.json).

### Planning boundaries

Native combined actions can replace some custom command construction, but the
observed interfaces do not produce one complete desired-state plan. Request
changes, missing or outdated versions, and removal of unwanted installations
must all be represented. A staged desired configuration can support native
upgrade previews without editing the live configuration first; its cleanup and
confirmation boundaries still need a design decision.

The existing arrays of version specs already map directly to native `[tools]`
arrays, without semantic translation or a schema change. Supporting the rest of
native `[tools]` would require changing the current `map[string][]string` model
and restricted version-spec validation. Mise also supports scalar strings,
options objects, and arrays of tool entries, including backend-specific options.
Passing through the whole native configuration is broader still, because it
includes settings and environment configuration. These alternatives are not
equivalent in scope.
[Native schema](https://github.com/jdx/mise/blob/main/schema/mise.json),
[current Go types](../../go/pkg/config/types.go),
[current validation](../../go/pkg/config/config.go)

Native pruning's protection of tracked project requirements also differs from
the existing planner's exclusive ownership of installed versions. Delegating
cleanup to `prune` would change that policy; these findings do not equate them.
[Prune documentation](https://mise.jdx.dev/cli/prune.html),
[current planner](../../go/pkg/mise/plan.go)

The linked upstream `main` sources are moving references and were inspected
alongside installed help. Where upstream docs and this executable differ, the
version-specific empirical results above describe what was actually observed.
No implementation change follows automatically from these findings.
