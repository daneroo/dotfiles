# Review: executable reconciliation plan (`thoughts/plans/reconfig-executable-plans.md`)

## 1. Overall assessment

The runner and confirmation design is sound and fits the codebase. The decisions you accepted hang together, and the two interfaces (`Reconciler.Plan` and `Plan.Report/Apply`) are a reasonable seam.

**The draft is not ready to implement as written.** Two parts are underspecified enough to cause wrong behavior on a live machine:

- **Mise:** how changes are applied and cleaned up.
- **Homebrew under `-y`:** whether one run actually converges.

There are also several hidden ways the current code kills the process outright (`log.Fatal`, a possible panic, `os.Exit`). These would defeat the runner's failure handling.

The Mise section is the weakest part. It adds a temporary global config and a native upgrade preview. Neither gives structured data the plan is allowed to use, and both create obligations about ordering, temp-file contents and parsing. A simpler approach is already half-built in `go/pkg/mise`.

Most fixes below are small. Two need a decision from you first (Homebrew convergence, and how Mise applies changes).

## 2. Material findings (highest severity first)

### F1 — Mise: the planned apply path can remove things outside the report, and its cleanup list can disagree with what actually gets installed *(correctness, high)*

Plan § Mise proposes: align requests with `use -g --fuzzy`, converge with `upgrade --global`, then run exclusive cleanup computed at plan time. Problems:

- **Unreported removals.**
  - `upgrade --global` schedules the old version for pruning after a grace period by default (research note, "upgrade --global … schedule `1.0.0` for pruning after 24 hours").
  - `unuse -g` prunes by default (research, `mise unuse`).
  - The current planner emits `mise unuse -g <tool>` and its comment is wrong (`go/pkg/mise/plan.go:80-87`).
  - Either way, Mise removes installations that are not shown as actions, or shown twice.
  - Concrete case: a stray `bun` has `1.4.3` installed. If Apply runs `unuse -g bun` before `uninstall bun@1.4.3`, the uninstall targets something already pruned and can fail. That would be a spurious execution failure that blocks all later applies. (Whether `mise uninstall` of a missing version exits non-zero is unverified.)
- **Hidden ordering.** `upgrade --global` only converges the *desired* requests if alignment has already been written to the live config. That obligation sits inside Apply and is not stated.
- **Cleanup drift.**
  - The plan admits that native previews can resolve again at execution, and that the preview is human text it must not parse.
  - So the report's "Would install X" and the cleanup list come from different sources. Neither one is the list of concrete versions Apply will act on.
  - If a release appears between plan and apply, the result is an extra leftover version (harmless) and a report that doesn't match what was done.

**Smallest fix:** make every removal an explicit, reported action, and turn native pruning off:

- use `use -g --remove` or `unuse -g --no-prune` for strays;
- use `upgrade --no-prune` if upgrade is kept;
- in Apply, run in this order: align, then install, then uninstall, then remove strays;
- skip cleanup if any earlier step failed.

Better still, see the alternative in §3, which installs the exact versions the report shows.

### F2 — Mise: the temporary "desired global config" costs more than it returns *(design or correctness, high)*

Plan § Mise says to build a temporary global config "by replacing tool declarations while preserving the relevant non-tool global settings":

- **It needs a TOML parser.** The repo has no TOML library (`go.mod` only requires `yaml.v3`). Rewriting the user's global file means either adding one, or copying the file and editing the copy with native commands (`config set` / `use --remove` aimed at the temp path). That second route is an unprobed workflow.
- **"The global config" may be several files.** Mise also reads `config.local.toml`, `conf.d/*.toml` and system config. Whether redirecting the global file (presumably `MISE_GLOBAL_CONFIG_FILE`) masks or merges those is unverified. The research only tested an isolated, single-file sandbox.
- **What it produces is limited.** The outputs are a human-readable preview and a pass/fail validation. The plan forbids scraping the preview and `outdated --json` is lossy. So the structured comparison (unchanged, missing, outdated, unwanted) must come from elsewhere anyway, which leads straight back to `mise latest` per request (`go/pkg/mise/reconcile.go:166-181`).
- **It adds temp-file obligations** (cleanup on error and cancellation) for what is mostly a display feature.

**Smallest fix:**

- Drop the staged config for v1.
- Narrow the supported shape to a scalar string or an ordered list of strings. The plan already defers options objects.
- Replace the regex check (`go/pkg/config/config.go:160-162, 238`; `config.schema.json:36`) with "non-empty string". Let Mise reject a bad request when `mise latest` runs during planning; that is already a planning error today. Mise still owns what a request means.

This **relies on a draft assumption**: that `mise latest tool@req` picks the same version `upgrade`/`use` would for the tools you use (bun `latest`, python prefixes, node `lts`). Probe it (see §4).

### F3 — Homebrew `-y` does not converge in one run, and the planned e2e would uninstall its own prerequisites *(correctness, high)*

- **Orphaned dependencies.** `Extraneous` returns only the minimal *roots* (`go/pkg/brewdeps/reconcile/extraneous.go:30, 37-60`).
  - Applying those uninstalls leaves the roots' formerly-required dependencies installed.
  - The next run then reports them as extraneous.
  - Plan § Homebrew packages says to keep root selection, while § Verification expects "`-y` convergence followed by an unchanged rerun". Those conflict whenever a removed root has dependencies.
  - `brew autoremove` is not a safe fill-in: it can remove a *desired* formula that was originally installed as a dependency.
- **The e2e would remove node and mise.** `e2e/test.sh:55` and `:80` install `node` and `mise` by hand, but `e2e/config.yaml` lists neither. A `-y` run would uninstall both as extraneous, after which npm and Mise both become missing tools.

**Decision needed** (see §4):

- (a) accept that removal takes more than one run, and say so in the report; or
- (b) uninstall the full extraneous set in a single `brew uninstall` call, while the report still groups it by root.

Either way, add `node` and `mise` to the e2e desired state (as `config.yaml` does under `bootstrap`), or use config variants for each step.

### F4 — Several current code paths kill the process, bypassing the runner *(correctness, medium-high)*

The plan assumes errors come back as values so the runner can switch to dry-run, keep planning, and set the exit code. Today:

- `log.Fatal` in `brewdeps/actual/actual.go:57-60` and `:99-107`, and in `extraneous.go:32`.
- `f.InstalledVersions[0]` in `brewdeps/actual/outdated.go:49-55` panics if Homebrew ever returns an empty list.
- `handleError` and the `os.Exit` paths in `go/cmd/reconfig/main.go:41-45, 86-95`.
- Code that prints directly instead of going through the renderer:
  - `tool.Installed` prints its own skip line (`go/pkg/tool/tool.go:12-20`);
  - `LoadConfig` prints (`config.go:49`);
  - brewdeps prints progress lines ("✓ - Got Installed").

**Smallest fix:**

- Convert these to returned errors.
- Give `tool.Installed` a non-printing form and report a missing tool as unavailable.
- Have the runner print each heading *before* calling `Plan`, so any remaining observation output lands under the right section.

Add this to step 2 of the plan's implementation sequence; it isn't in it now.

### F5 — Cancellation and terminal input are underspecified *(correctness, medium)*

- **Hard kills mid-install.** `exec.CommandContext` sends SIGKILL by default when the context is cancelled. On Ctrl-C, that can hard-kill `brew upgrade`, `mise install` or Stow in the middle of an operation. Fix: set `cmd.Cancel` to send SIGINT or SIGTERM, plus a `WaitDelay`.
- **Defers must run.** Use `signal.NotifyContext`, so planning's temp-file cleanup (if any remains after F2) still runs.
- **Input handling:**
  - A Ctrl-C while the confirmation prompt is waiting must cancel the run, not count as a decline.
  - Write prompts to stderr or `/dev/tty`, so `./reconfig.sh | tee log` still shows them.
  - Apply subprocesses need `Stdin` connected, because cask installs may ask for a sudo password. The plan says authentication stays native, but this wiring isn't stated.
- **Non-interactive detection** should check that stdin is a terminal. With that, `e2e/run.sh:15` (`docker run` without `-i`) falls back to dry-run, which is correct. The e2e should assert the fallback notice.

### F6 — Stow: an empty simulation is not empty output, and the Go program now depends on the repo root *(correctness, medium)*

- **No-op detection needs a small parser.**
  - In simulation mode (`-n`), Stow always prints a "simulation mode" warning on stderr (from upstream behavior; not verified against your installed version).
  - So "no output" can't mean "nothing to do". Deciding unchanged versus actionable means classifying `LINK:`, `UNLINK:`, `MKDIR:` and `RMDIR:` lines and ignoring that warning.
  - That parser is small and the format is stable, but it should be named explicitly, with fixture tests, rather than hidden inside "capture stderr as preview".
  - Conflicts show up as a non-zero exit plus a `WARNING! … conflicts` message. Use the exit status, not the text.
- **The repo root becomes an implicit input.**
  - Stow (`installDotLinks.sh:26`) and the completion paths (`main.go:69-72`, relative `./core/...`) both assume the program's working directory is the repo root.
  - Once Stow moves into Go, that working directory becomes part of the interface. Resolve the root once and pass it when building the reconcilers.
- **E2E will now exercise Stow.** The e2e calls the binary directly and the container has no `stow`. The Stow reconciler will report it as a missing tool. That's fine, but assert it.

### F7 — Completions: the Bun premise is contradicted inside the plan, and ordering with Stow is unverified *(feasibility, medium)*

- **The Bun premise contradicts itself.**
  - The current code runs `bun completions` through `CombinedOutput()` (`go/pkg/completions/completions.go:92`), so stdout is a pipe, not a terminal.
  - The plan itself says Bun's source has a non-terminal stdout path. If that's right, today's code prints the script to a buffer and throws it away, then patches whatever `bun.completion.bash` was last written there, which may be stale.
  - The plan's claim that Bun "writes a Homebrew completion file before it can be read" then only holds for a terminal.
  - **This rests on the plan's reading of the upstream source, not on observed behavior.**
- **If the stdout path is real,** planning becomes `Output()` (stdout only; `CombinedOutput` would mix stderr into the file), followed by the in-memory patch.
- **One planning side effect may remain:** the bunx symlink. For a Mise-installed bun, "beside its executable" is inside Mise's install tree, so planning would write there. Confirm that in a sandbox or container.
- **Stow ordering:** completion files are written into `core/`, which Stow manages. A *new* completion file only appears in `~` after a later Stow run, unless `~/.config/bash_includes` is a single folded directory link. That's unverified here, and it's the same as today, so it's not a regression. But the proposed order (Stow, then completions) doesn't fix it, and the plan claims ordering benefits.

### F8 — Runner rules left ambiguous *(correctness, medium-low)*

- **Planning errors.** Plan § Intended behavior switches to dry-run only after a decline or an *apply* failure. Yet § Stow says a conflict is a planning error after which later reconcilers are "inspected in dry-run".
  - Decide whether a real planning error also stops later applies. I recommend yes: one rule, "any real error ends applying for the rest of this run".
- **Report states need definitions.** Define how they map to exit status:
  - unchanged → 0;
  - actionable but not applied → 0;
  - unavailable (missing tool or unmet prerequisite) → 0;
  - error → non-zero.
- **E2E coverage gaps:**
  - check that `-n` changes nothing, including tracked completion files in the container's repo copy (`git diff --quiet`);
  - check that `-y` handles a Mise stray;
  - check the fallback notice when input isn't a terminal.

### Optional preferences (not correctness)

- **The pending-upgrade gate hides the package plan.** Outdated casks such as `claude-code@latest` appear almost daily, so the package plan would almost always show as unavailable in `-n` and after a decline. Only removals need the trusted dependency graph. Missing installs could still be shown, with removals marked unavailable.
- **`Plan` could be a struct instead of an interface.** For example `Plan{Report; Apply func(ctx) error}`, where a nil `Apply` means not actionable. It's slightly less ceremony; the trade-off is minor either way.
- **Glossary.** "Report" and "unavailable" are not in `GLOSSARY.md` and should be added if kept.

## 3. Alternative Mise design (materially simpler)

Keep the runner interface as drafted. Change only how Mise works inside, reusing code that already exists and is tested.

- **Observation:** unchanged.
  - `mise ls --installed --json`;
  - ordered global requests via `ls --global` plus `config get` (`reconcile.go:98-162`);
  - `mise latest tool@request` for each request (handles multiple requests per tool, since it runs per request).
- **Comparison:** unchanged. The existing `plan()` (`go/pkg/mise/plan.go`) already covers missing/outdated, alignment, exclusive cleanup and strays, with tests.
- **Actions:** turn the printed strings into argument lists, applied in this order:
  1. Align requests: `use -g --fuzzy tool@req…`, or the config-only `config set` route if a probe confirms it works for lists.
  2. `mise install tool@<concrete version shown in the report>`.
  3. `mise uninstall tool@<version>`.
  4. Remove strays with `use -g --remove` or `unuse -g --no-prune`.
  5. Stop at the first failure; never clean up after a failed install.
- **No temporary files, no TOML handling, no reading of preview text.**

**Trade-off:**

- You give up the native upgrade preview text and lean on `mise latest` matching what Mise would select.
- The comparison stays in reconfig (it already is). Mise still decides what each request means.
- A mismatch would show up as a non-converging rerun in e2e, not as silent damage.
- Options objects stay unsupported until there's a faithful round-trip for them, which the plan already defers.

## 4. Decisions and feasibility probes needed

Run all probes in an isolated sandbox or the e2e container, never on this machine.

1. **Mise resolution:** for bun `latest`, python `3.14`, and node `lts` (if used), does `mise latest` equal what `upgrade --global --dry-run` / `use` would install?
2. **Mise commands:**
   - does `config set` with a list type write `tools.X` correctly?
   - confirm `use -g --remove` doesn't prune on the installed version;
   - what exit code does `mise uninstall` give for a version that isn't installed?
3. **Homebrew convergence:** choose (a) multiple runs or (b) a full-set uninstall (F3). For (b), check that one `brew uninstall a b` call accepts a set where some packages depend on others.
4. **Bun:** in a container, does `bun completions | cat` print the script to stdout, and what does it write (bunx symlink, Homebrew-prefix file)?
5. **Stow:** check the simulation output grammar on the installed version, and whether `~/.config/bash_includes` is a folded directory link.
6. **Runner rule:** does a real planning error end applies for the rest of the run (F8)?

## 5. Retain

- One shared runner that owns mode, confirmation, rendering and the exit code; reconcilers never prompt.
- Observing one reconciler at a time against the state actually left behind, with no precomputed machine state.
- Plans that hold argument lists and file contents, never shell text to `eval`; the e2e no longer extracts printed commands.
- The rule that planning cleans up its own scratch files before returning.
- Installs before removals; stopping a plan at its first failure; never running `brew cleanup` after a failed upgrade.
- One Homebrew reconciler with formulae and casks grouped separately; one Mise reconciler; exclusive cleanup policy stated explicitly.
- Not using `outdated --json`, and not treating native prune as equivalent to exclusive cleanup.
- The read-only dead-link check; Stow-missing reported as a missing tool.
- Honestly flagging Bun as a feasibility item rather than calling a managed write read-only.
