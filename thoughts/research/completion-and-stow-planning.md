# Completion and Stow planning

Date: 2026-10-20. Follow-up to findings F6 and F7 in the
[Claude design review](../reviews/reconfig-claude-design-review.md).

## Evidence and isolation

Installed versions: Bun **1.4.3**, GNU Stow **2.4.1**. The observations below
come from temporary fixtures, not execution against the machine's managed
dotfiles or completion targets. The retained probe ran under
`/tmp/reconfig-completion-stow-55hgn5tg/`; its
[`results.json`](/tmp/reconfig-completion-stow-55hgn5tg/results.json) records
commands, output and filesystem changes. An initial identical probe used the
system temporary directory before the retained probe was repeated under
`/tmp`. Neither probe applied live configuration changes.

For Bun, the actual installed executable was copied into scratch. A harmless
executable `bunx` fixture appeared first on its controlled PATH. The working
directory, `BUN_INSTALL`, XDG config/data/state/cache locations, temporary
directory and Homebrew-prefix fixture were all under scratch. Existing scratch
Bash completion directories contained sentinel files. `SHELL` was `/bin/bash`;
stdout and stderr were separate pipes. `HOME` retained its original value and
was not repurposed. The copied executable confines the executable-adjacent
symlink fallback to scratch even if the PATH fixture is not found.

For Stow, each scenario had its own scratch repository, `core` package and
target directory. Every invocation used `--simulate --verbose`, explicit
`--dir` and `--target`, and a scratch working directory. No real Stow apply
operation ran. Snapshots compared file bytes, directory entries and symlink
targets before and after each probe; access times were not compared.

## Bun stdout generation works

Empirical command: the copied `bun completions`, with the environment above.

| Observation | Result |
| --- | --- |
| Exit code | 0 |
| Standard output | 11,565 bytes of Bash completion script |
| Standard error | Empty |
| Scratch filesystem changes during command | None |
| Existing regex-patch marker | Present once |
| Output SHA-256 | `3e872ca6ce5a82989a51348ada063b425a75983992f100a9ee8173d81727d505` |

The captured script and stderr were saved after the before/after comparison,
as `bun-stdout.bash` and `bun-stderr.txt` beside the result record.

This confirms the non-terminal stdout behavior for the installed executable.
Upstream source selects the shell from `SHELL`, and returns the embedded script
through stdout before choosing or writing a completion directory. However, its
earlier `install_bunx_symlink` call can create a link beside the Bun executable
when `bunx` is absent from PATH. Piping alone therefore does not establish that
the operation has no filesystem side effects. Scratch `HOME` alone would not
confine that executable-adjacent write either.
[Bun completion installer](https://github.com/oven-sh/bun/blob/main/src/runtime/cli/install_completions_command.rs)

The controlled PATH and executable copy avoided that hazard in the probes.
This is an observed property of this version and setup, not a guarantee that
future Bun versions cannot introduce new side effects. Implementation should
retain an isolated fixture test and handle failed generation as an error.

### Correction to the existing code's premise

The current function runs `bun completions` using `CombinedOutput`, discards
the successful output, then reads the Homebrew file `bun.completion.bash` and
patches that file's contents (`go/pkg/completions/completions.go:92-105`).
`CombinedOutput` makes stdout non-terminal, matching the probed behavior.
Consequently, this call does not refresh the Homebrew source file as the
current comment assumes: it obtains a fresh script in memory and ignores it.
The subsequently read file may be stale, or absent. The probes do not establish
whether the machine's existing source file currently differs from fresh output;
no such live-content comparison was made.
[Current implementation](../../go/pkg/completions/completions.go)

A feasible plan can capture **stdout separately**, apply the existing patch to
those bytes, compare them with the intended target, and defer the target write
to execution. `CombinedOutput` is unsuitable for persisted script content
because diagnostics would be mixed into the bytes. Generation must retain the
isolation described above; an ordinary piped call against the live executable
would reintroduce the `bunx` side-effect risk.

## Stow simulation distinguishes the three cases

Each package contained one `.reconfig-fixture` file. The target either had no
entry, the correct relative symlink, or a conflicting ordinary file. The
operation was:

```sh
stow --simulate --verbose --dir <scratch-repo> --target <scratch-home> core
```

| Target state | Exit | Standard output | Standard error |
| --- | --- | --- | --- |
| Link needed | 0 | Empty | One `LINK:` operation, then simulation warning |
| Correct link already present | 0 | Empty | Simulation warning only |
| Conflicting ordinary file | 1 | Empty | Conflict explanation and operations-aborted message |

No scenario changed its filesystem state. The warning-only unchanged case
confirms F6: nonempty stderr is not proof that Stow has work to apply. The
nonzero conflict status is distinct from an unchanged or actionable preview.

Installed source plans operations, checks conflicts, exits 1 on conflict, and
returns without processing tasks when simulation is enabled. Verbosity 1 emits
operation lines; higher verbosity adds diagnostic detail. A small classifier
for the operation lines must account for `LINK`, `UNLINK`, `MKDIR`, `RMDIR` and
any other operation supported by the selected command, while disregarding the
simulation warning. Native preview text is for display and classification;
execution should rerun the retained Stow arguments without `--simulate`, rather
than execute text extracted from output.
[Official CLI source documentation](https://github.com/aspiers/stow/blob/master/_autodocs/api-reference/stow-cli.md),
[official verbosity documentation](https://github.com/aspiers/stow/blob/master/_autodocs/api-reference/Stow-Util.md)

The fixture establishes `LINK` and unchanged/conflict behavior, not every
operation grammar or nested tree-folding case. Additional fixture coverage is
needed for whatever operation classifier is implemented. Both Stow and the
completion destinations need an explicit repository root, rather than relying
on the caller's working directory.

## Ordering limit

Stow currently runs before completion generation. A newly created file under
`core/.config/bash_includes` reaches the home directory immediately only when
the target directory is already linked as a whole; individually linked files
require another Stow operation for a new file. A metadata-only check found that
this machine currently has `~/.config/bash_includes` linked to
`../.dotfiles/core/.config/bash_includes`, so its existing folded-directory
layout exposes new files immediately. No home-directory file contents were
read for this check. The fixtures and this machine's layout do not establish
that a single ordering handles every possible target layout. Preserve this
distinction when describing any ordering benefit.
[Current Stow invocation](../../installDotLinks.sh),
[current completion targets](../../go/cmd/reconfig/main.go)

These findings validate F6's unchanged-output caveat and F7's stdout/stale-source
concern. They provide a feasible isolated Bun generation path; they do not
authorize live writes or settle the remaining execution design.
