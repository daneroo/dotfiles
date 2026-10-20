# Homebrew executable removals

Date: 2026-10-20. Read-only source inspection supporting Claude review F3.
No live installation, update, uninstall, or Docker operation was performed.

## Observed behavior

Sources below are pinned to the installed Homebrew revision
`332963d53699d6a49b9d16bcb04b9fbc66ae5f00`.

- A single uninstall can include a formula and its dependencies. The dependent
  check excludes formulae and casks included in the removal set. An upstream
  test explicitly covers removing a dependent and dependency together.
  [Guard](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/installed_dependents.rb#L18-L45),
  [test](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/test/installed_dependents_spec.rb#L140-L146).
- Casks can depend on other casks. Homebrew models these as requirements,
  separately from formula dependencies. Its guard likewise allows dependent
  casks to be removed together with their dependencies.
  [Model](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/cask_dependent.rb#L46-L78),
  [test](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/test/cask/uninstall_spec.rb#L300-L313).
- Current `brew uninstall` automatically invokes `Cleanup.autoremove`, unless
  `HOMEBREW_NO_AUTOREMOVE=1` is set. Therefore Claude F3's statement that removing
  only roots necessarily leaves their orphaned dependencies is incomplete.
  [Uninstall](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/cmd/uninstall.rb#L129-L136).
- Autoremove uses installed dependency relationships and installation history,
  including `installed_on_request == false`; it does not know reconfig's desired
  state. A desired formula originally installed as a dependency can therefore
  be removed when no remaining installed package requires it. Conversely,
  explicitly requested extraneous dependencies can remain.
  [Selection](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/utils/autoremove.rb#L103-L120),
  [application](https://github.com/Homebrew/brew/blob/332963d53699d6a49b9d16bcb04b9fbc66ae5f00/Library/Homebrew/cleanup.rb#L1042-L1088).

## Current reconfig gaps

- [Extraneous](../../go/pkg/brewdeps/reconcile/extraneous.go) computes all extras
  but returns only `minimizeExtraneous` roots. That subset is unsuitable as the
  complete explicit execution set.
- [Actual observation](../../go/pkg/brewdeps/actual/actual.go) obtains dependency
  entries only for installed roots via `brew deps --installed`. It also assumes
  all dependencies are formulae and that casks cannot depend on casks.
- [IsTransitiveDep](../../go/pkg/brewdeps/reconcile/graph.go) has no dependency
  closure for a missing desired root. If desired A is absent and installed B is
  needed by A, B can be classified as extraneous. After installing A, Homebrew's
  normal guard rejects uninstalling B: the removal is prevented, but the plan
  was misleading and the run fails unnecessarily.
- [E2e config](../../e2e/config.yaml) omits node and mise, although the
  [scenario](../../e2e/test.sh) installs them later. Executing the full package
  plan would remove those prerequisites. Initial tests deliberately require
  them absent, so adding them unconditionally also changes that scenario.

## Proposed implementation choices

These are recommendations, not additional observed tool behavior:

1. Protect dependencies of every desired root, including missing roots, before
   computing removals. Alternatively, revalidate approved removals after
   installs and only drop removals that have become required. Do not add newly
   discovered removals without including them in an approved plan.
2. Execute the complete extraneous set. Minimal roots can remain a presentation
   aid.
3. Install first. Then remove all extra casks in one explicitly typed batch,
   followed by all extra formulae in one explicitly typed batch. Retain native
   dependency guards; do not use `--ignore-dependencies`. Stop on failure.
4. Set `HOMEBREW_NO_AUTOREMOVE=1` for managed uninstall and cleanup operations,
   making package removals explicit and reviewable.
5. Account for cask requirements in dependency protection. Separate cask and
   formula presentation does not justify separate ownership of the graph.
6. Use staged e2e desired configurations, or otherwise include node and mise in
   desired state once those prerequisites are installed.

Native operations are not transactional. Pins can prevent removal, multiple
formula keg versions can remain without `--force`, and cask uninstall can
partially fail. Convergence claims must allow correctly reported native
failures rather than imply unconditional success.
