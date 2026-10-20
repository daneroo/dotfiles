# Executable reconciliation plans

Published as [GitHub issue #11](https://github.com/daneroo/dotfiles/issues/11),
labelled `ready-for-agent`. The approved ticket breakdown is published as its
sub-issues #12 through #18.

## Problem Statement

Reconfig compares the machine's desired state with its actual state, but its
output mixes diagnostic reporting, repeated command alternatives, and changes
that happen immediately. Package changes generally require copying commands;
dotfile linking and completion writes already happen during the run. Pending
Homebrew upgrades stop the whole program before independent reconcilers report.

The machine owner needs a consistent way to review and apply changes one
reconciler at a time, preview the whole run, or apply all actionable plans. The
existing readable, Markdown-like presentation should be preserved. Implementing
this should simplify and unify the modules while retaining tested comparison
logic and native package-manager behavior where appropriate.

## Solution

Give every reconciler a common planning/execution contract and put presentation,
confirmation, mode transitions, and final exit status in one shared runner.
Observe and plan each reconciler against the actual state left by earlier work,
then show its report and optionally apply its plan.

Offer three modes: confirmation per reconciler by default; `-n, --dry-run` to
preview without managed changes; and `-y, --yolo` to apply without reconfig
confirmation. Default noninteractive invocation announces a dry-run fallback.
Declining a reconciler or encountering a real error ends applying for that run
but continues useful later observation. Preserve normal upstream metadata
refresh and cache behavior during observation.

## User Stories

1. As the machine owner, I want desired and actual state compared before changes
   are applied, so that I can review how the machine will converge.
2. As the machine owner, I want one confirmation per actionable reconciler, so
   that approval is useful without becoming a sequence of per-package prompts.
3. As the machine owner, I want explicit dry-run mode, so that I can inspect
   plans without installs, removals, selection edits, linking, or completion writes.
4. As the machine owner, I want explicit YOLO mode, so that an approved automated
   run can apply all actionable plans without reconfig prompts.
5. As a script author, I want default noninteractive execution to announce and
   use dry-run, so that the command does not hang waiting for confirmation.
6. As a CLI user, I want conventional short and double-dash long options and a
   stable program name in help, so that the command behaves predictably.
7. As the machine owner, I want the existing Markdown-like headings, symbols,
   indentation, and bootstrap style preserved, so that output stays familiar.
8. As the machine owner, I want each intended operation presented once, so that
   duplicate individual and batched command alternatives do not obscure the plan.
9. As the machine owner, I want a decline to switch all following reconcilers to
   dry-run, so that no later managed changes happen in that invocation.
10. As the machine owner, I want real errors to stop subsequent applies while
    allowing useful later reports, so that I can see remaining work in one run.
11. As the machine owner, I want missing tools and unmet prerequisites reported
    locally, so that independent reconcilers can still be inspected.
12. As the machine owner, I want later observations to see earlier successful
    changes, so that newly installed tools can be used in the same run.
13. As the machine owner, I want pending Homebrew upgrades reported and
    executable without ejecting from the whole program, so that I can still
    inspect independent reconcilers after declining them.
14. As the machine owner, I want formulae and casks displayed separately inside
    one Homebrew package reconciler, so that presentation stays clear and
    dependency planning remains coherent.
15. As the machine owner, I want all desired dependencies protected, including
    those of missing desired packages and dependent casks, so that cleanup does
    not remove something required by the desired state.
16. As the machine owner, I want the full unwanted Homebrew package set reported
    explicitly, so that native automatic removal cannot silently remove a
    desired package based on its installation history.
17. As the machine owner, I want a successful supported run to converge package
    membership in one invocation, so that an unchanged rerun is meaningful.
18. As the machine owner, I want Stow conflicts and proposed link operations
    visible before application, so that preview does not change my dotfiles.
19. As the machine owner, I want newly generated completion files linked during
    the same successful run, so that correctness does not depend on directory
    folding or an extra invocation.
20. As the machine owner, I want Mise to interpret native version expressions,
    so that reconfig does not implement a competing version grammar.
21. As the machine owner, I want multiple Mise requests kept in their declared
    order, so that the first request continues to define the default.
22. As the machine owner, I want installation targets to match the concrete
    versions reported before approval, so that apply does not resolve a different
    release while cleanup still uses the old plan.
23. As the machine owner, I want Mise's global requests aligned while preserving
    unrelated settings and project configuration, so that ownership is explicit.
24. As the machine owner, I want exclusive Mise installation cleanup for now,
    so that versions outside the desired resolutions are reported and removed.
25. As the machine owner, I want npm global installs, updates, and removals to
    use the same review and execution flow as other reconcilers.
26. As the machine owner, I want completion content compared before writing,
    including fresh Bun output and its existing fixes, so that previews do not
    mutate managed files or patch stale content.
27. As the machine owner, I want cancellation to stop the run promptly and
    gracefully, so that Ctrl-C is not treated as declining just one reconciler.
28. As the machine owner, I want native authentication and privilege handling
    retained during execution, so that package-manager requirements still work.
29. As a script author, I want pending, declined, and unavailable work separated
    from real errors in the exit status, so that automation can interpret results.
30. As a maintainer, I want presentation independent of execution and comparison,
    so that future output improvements do not require rewriting reconcilers.

## Implementation Decisions

- Introduce a shared runner and a real Go reconciler interface. A reconciler has
  a stable name and a planning operation; its returned plan exposes report data
  and an apply operation. Bind desired configuration and process dependencies
  when constructing the implementation. Keep domain-specific observation and
  ordering inside the implementation.
- The report distinguishes actionable, unchanged, and unavailable plans. The
  runner separately records previewed, declined, applied, and failed outcomes.
  Unchanged and unavailable plans never prompt or apply.
- The runner alone owns confirmation, rendering, execution mode, and the final
  exit status. Reconcilers do not prompt or receive confirmation-mode options.
  Observe each reconciler after the preceding one finishes; do not precompute a
  hypothetical complete machine state.
- A decline changes the effective mode to dry-run for the rest of the run. Any
  real planning or execution error does the same and makes the final exit status
  non-zero. Missing tools, unmet prerequisites, pending changes, and declines
  alone do not cause a non-zero status. A fresh invocation resets the mode.
- Cancellation stops the run. Use signal-aware cancellation, graceful child
  process interruption with a bounded wait, and cleanup on all paths. Return
  errors from helpers instead of terminating the process internally. Preserve
  native subprocess input and output for authentication and diagnostics.
- Keep the standard Go flag parser. Customize help to group short and long
  aliases, use a stable program name, retain configuration and verbose options,
  and add dry-run and YOLO options. Reject conflicting modes. Both explicit modes
  bypass reconfig confirmation; native authentication remains separate. Do not
  add an external CLI dependency or global mutable execution mode.
- Bootstrap keeps its existing prerequisite checks and presentation, forwards
  arguments to the Go program, and delegates managed linking to the runner.
  Preserve existing headings, symbols, indentation, and useful diagnostics.
  Remove duplicate execution alternatives rather than redesigning presentation.
- Normal upstream metadata refreshes and cache writes are acceptable during
  observation. All managed installations, removals, default edits, links, and
  completion writes require apply. Plans retain executable arguments and file
  contents; displayed output is never evaluated as shell code. Temporary
  resources stay internal to implementations and are cleaned on every path.
- Run Homebrew upgrades, Homebrew packages, Mise, npm globals, completions, and
  Stow in dependency-aware sequence. Stow follows completion generation so new
  files are linked in the same successful run even with unfolded directories.
  Resolve newly installed tools through their manager when their executable
  locations are not yet present in the parent shell's environment.
- Homebrew upgrades observe metadata and outdated packages, then apply upgrade
  followed by cleanup. A failed upgrade prevents cleanup. Pending upgrades make
  dependency-sensitive package planning unavailable without preventing unrelated
  later observations; successful upgrades are followed by fresh observation.
- Keep formulae and casks in one package reconciler with shared dependency
  planning and separate display groups. Protect desired packages and all their
  dependencies, including missing desired roots and cask-to-cask requirements.
  Report the complete unwanted set; minimal roots may help explain it but are
  not the execution set.
- Disable Homebrew automatic removal during managed mutations and cleanup.
  Apply installs first, then unwanted casks as a set, then unwanted formulae as
  a set, preserving native dependency guards. Revalidation may drop newly
  required removals but must not introduce unapproved removals. Do not force
  removal by ignoring dependency checks.
- Reuse the tested Mise observation and comparison approach. Support scalar
  strings and nonempty ordered lists of distinct strings. Keep version strings
  opaque, remove the restricted expression regex, and ask Mise to resolve each
  request. Empty native resolution output is an error even if its exit status
  is zero. Preserve the first request as the default and normalize equivalent
  singleton representations.
- Do not compose a temporary global TOML configuration or add a TOML parser for
  this version. Mise's persistent global file stays Mise-managed outside Stow.
  Preserve unrelated global settings and do not edit project configuration.
  Existing Stow-managed shell and completion setup remains applicable.
- Mise apply installs the approved concrete versions, aligns global requests
  through native configuration-only edits, explicitly uninstalls unwanted
  concrete versions, and removes stray global requests with pruning disabled.
  Stop at the first failure and never clean up after a failed installation or
  request update. Native literal string assignment and appending preserve list
  order without splitting opaque expressions on commas.
- Retain exclusive Mise cleanup for the initial version: installations outside
  the desired resolved versions are unwanted. Native project-aware pruning is
  a different policy and is not substituted. Do not infer historical ownership
  from current configuration sources, scrape human previews, or use the lossy
  multi-request outdated JSON as a complete plan.
- npm keeps its existing global desired/actual/outdated comparisons and applies
  installs or updates before removals. A missing manager is unavailable.
- Completion planning captures desired content without managed writes, compares
  existing content, and applies only changed targets. Capture fresh Bun stdout
  separately from diagnostics, apply its existing corrections in memory, and
  isolate the generator's preliminary bunx behavior. Keep tool discovery and
  scratch ownership internal.
- Stow uses native simulation and conflict checking for planning. Distinguish
  actual operation lines from the simulation warning emitted for an unchanged
  result; use exit status for conflicts. Apply real linking only after approval.
  Keep the dead-link diagnostic read-only and resolve repository/home targets
  explicitly rather than relying on accidental working-directory assumptions.
- Keep partial failures visible; automatic rollback is not required. Successful
  convergence is subject to native tool errors and supported ordinary package
  states, not a promise to override pins or bypass native guards.

## Testing Decisions

- The primary test seam is the shared runner operating real reconciler plans
  with injected process/filesystem dependencies. The shared interface and its
  behavioral guarantees were explicitly reviewed and accepted during design.
  Prefer observable reports, prompts, applied changes, and exit results over
  assertions on private helpers or duplicated implementation logic.
- Exercise all modes, conflicting flags, conventional help, noninteractive
  fallback, unchanged/unavailable plans, acceptance, decline, planning errors,
  execution failures, and cancellation. Verify that a mode transition prevents
  every subsequent apply while permitting independent later planning.
- Verify sequential observation sees earlier actual changes. Verify no managed
  writes during dry-run, while allowing documented metadata/cache effects.
  Exercise graceful process interruption and cleanup rather than just accepting
  a context argument that never reaches subprocesses.
- Retain useful existing comparison and configuration tests; adapt them at the
  new public seam where behavior moves. Existing Mise planning cases and the
  container scenario provide prior art. Replace obsolete print-only assertions
  once equivalent public behavior is covered.
- Cover Homebrew's full removal set, desired dependencies introduced by missing
  roots, desired formulae originally installed as dependencies, cask requirements,
  disabled automatic removal, native guards, and incomplete graph/error cases.
- Cover Mise missing/current/outdated/exact requests, ordered multiple requests,
  native expressions, empty resolution output, scalars/lists, comma-containing
  expressions, changed defaults, strays, exclusive cleanup, preservation of
  unrelated global settings, and a release appearing between plan and apply.
- Cover Stow unchanged/needed/conflicting simulations, missing tools, fresh
  completion stdout, Bun corrections, changed/unchanged files, and both folded
  and unfolded target-directory layouts.
- Update the container scenario to use explicit dry-run for previews and YOLO
  for convergence followed by an unchanged rerun. Declare the scenario's own
  prerequisites in staged desired configurations. Assert noninteractive fallback
  and dry-run preservation of generated repository files. Stop extracting and
  evaluating printed commands as the execution mechanism.
- Run the repository quality gate and container E2E checks when available.
  Validate with controlled adapters, temporary directories, and containers;
  do not run live reconciliation on the owner's machine without asking.

## Out of Scope

- Per-action confirmation or selection, a TUI, or a broad presentation redesign.
- An external CLI framework or new TOML parsing dependency for Mise planning.
- A generated persistent or Stow-managed Mise global configuration.
- Arbitrary native Mise option objects, the whole native configuration surface,
  project configuration management, or project-aware installation ownership.
- Replacing native managers, overriding native dependency guards, automatic
  rollback, or promising successful convergence despite native failures.
- Implementing the feature or changing the live machine as part of writing this
  spec and its implementation tickets.

## Further Notes

An independent read-only Claude codebase-design review supported the shared
runner and identified native cleanup, process-exit, cancellation, and planning
gaps. The accepted revisions simplify Mise around existing tested logic and
make Homebrew cleanup explicit. Follow-up probes and source inspection checked
the material findings; the review is input to the agreed design, not a separate
source of product requirements.

Isolated Mise probes verified native configuration-only edits, literal ordered
string handling, preservation of unrelated settings, and concrete installations
that remain fixed when a newer release appears. Bun and Stow probes verified
stdout completion generation and unchanged/needed/conflict simulation cases.
These observations used controlled fixtures and installed versions; they do not
claim support for every native backend or future version.

Current Homebrew source confirms that uninstall can run automatic removal and
that dependencies included in the same removal set are supported. Its automatic
cleanup follows native installation history rather than this desired state.

Primary references: [Mise config-only editing](https://mise.jdx.dev/cli/config/set.html),
[Mise explicit installation](https://mise.jdx.dev/cli/install.html),
[Mise removal without pruning](https://mise.jdx.dev/cli/unuse.html),
[Homebrew uninstall behavior](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/cmd/uninstall.rb),
and [Homebrew removal-set dependency handling](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/installed_dependents.rb).
