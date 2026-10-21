#!/usr/bin/env bash
# The e2e scenario, run inside a fresh Homebrew container (see run.sh).
# It reconciles against the staged desired states e2e/config.yaml and
# e2e/config.stray.yaml, and asserts output, exit codes, and the resulting
# machine state at every step. reconfig applies its own plans: nothing it
# prints is run.
set -uo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/.." && pwd)"
cd "$REPO_ROOT"

CONFIG=e2e/config.yaml
STRAY_CONFIG=e2e/config.stray.yaml
DESIRED_FORMULAE="hello jq mise node stow tree"
EXTRANEOUS_FORMULA=figlet
NPM_GLOBAL=semver
STRAY_TOOL=jq
STRAY_VERSION=1.7.1
# A completion file reconfig generates into the repository; removed first,
# so it is new and Stow must link it in the same run.
GENERATED=core/.config/bash_includes/mise_completion.bash
BUN_COMPLETION="$(brew --prefix)/share/bash-completion/completions/bun"
# core/.npmrc moves npm's global prefix here once Stow links it; set from the
# start, so npm globals stay in one place, as on a machine already linked.
export NPM_CONFIG_PREFIX="$HOME/.npm-global"

echo "Container: $(uname -sm), $(brew --version | head -1), bash ${BASH_VERSION}"

main() {
    local scenario_start=$EPOCHREALTIME

    step "0. Setup: a new generated file, and an unfolded ~/.config/bash_includes"
    rm "$GENERATED" || fail "cannot remove $GENERATED"
    mkdir -p ~/.config/bash_includes
    echo "# not managed by reconfig" >~/.config/bash_includes/local.sh
    echo "✓ removed $GENERATED; ~/.config/bash_includes is a directory, not a link"
    local fresh
    fresh=$(snapshot)

    step "1. Help, and conflicting modes are a usage error"
    run_reconfig --help
    assert_exit 0
    assert_output "Usage: reconfig [options]"
    run_reconfig -n -y
    assert_exit 2
    assert_output "--dry-run and --yolo cannot be used together"

    step "2. An explicit dry run on a fresh machine shows the plans and changes nothing"
    run_reconfig --dry-run -c "$CONFIG"
    assert_exit 0
    assert_output " - mode: dry-run"
    local install_line
    install_line=$(grep '^ HOMEBREW_NO_AUTOREMOVE=1 brew install --formula ' "$OUT" | tail -1)
    assert_equal "missing formulae" "$DESIRED_FORMULAE" "$(words_sorted "${install_line# HOMEBREW_NO_AUTOREMOVE=1 brew install --formula }")"
    assert_section_output "Homebrew Packages Section" "△ - Dry run: not applied"
    assert_section_output "Mise Section" "✗ - mise is not installed, skipping"
    assert_section_output "NPM Globals Section" "✗ - npm is not installed, skipping"
    assert_section_output "Stow Section" "✗ - stow is not installed, skipping"
    assert_none_applied
    assert_unchanged "$fresh"

    step "3. Without a terminal, the default mode previews only, even when input says y"
    run_reconfig_with_input y -c "$CONFIG"
    assert_exit 0
    assert_output "△ - Input is not interactive: previewing only, as with --dry-run"
    assert_output " - mode: confirm"
    assert_none_applied
    assert_unchanged "$fresh"

    step "4. YOLO installs the prerequisites and uses them; existing dotfiles conflict with Stow"
    run_reconfig --yolo -c "$CONFIG"
    assert_exit 1
    for name in "Homebrew Packages" "Mise" "NPM Globals" "CLI Completions"; do
        assert_output "✓ - $name applied"
    done
    assert_section_output "Stow Section" "✗ - planning failed: stow cannot link core"
    assert_output "over existing target .bashrc"
    for f in $DESIRED_FORMULAE; do
        brew list --formula "$f" >/dev/null 2>&1 || fail "$f is not installed"
    done
    echo "✓ installed: $DESIRED_FORMULAE"
    bun_installed || fail "bun is not installed"
    echo "✓ bun installed by mise"
    npm_global_installed || fail "npm global $NPM_GLOBAL is not installed"
    echo "✓ npm global installed: $NPM_GLOBAL"
    [ -f "$GENERATED" ] || fail "$GENERATED was not generated"
    echo "✓ generated: $GENERATED"
    assert_bun_completion
    [ -L ~/.bashrc ] && fail "~/.bashrc was replaced despite the conflict"
    echo "✓ ~/.bashrc left as it was"

    step "5. With the conflicting dotfiles moved aside, YOLO links core, including the new file"
    mv ~/.bashrc ~/.bashrc.orig && mv ~/.profile ~/.profile.orig || fail "cannot move dotfiles aside"
    run_reconfig -y -c "$CONFIG"
    assert_exit 0
    assert_output_line " LINK: .config/bash_includes/mise_completion.bash => ../../.dotfiles/core/.config/bash_includes/mise_completion.bash"
    assert_output "✓ - Stow applied"
    assert_resolves ~/.bashrc "$REPO_ROOT/core/.bashrc"
    assert_resolves ~/.config/bash_includes/mise_completion.bash "$REPO_ROOT/$GENERATED"
    [ -L ~/.config/bash_includes/local.sh ] && fail "local.sh became a link"
    echo "✓ ~/.config/bash_includes/local.sh left as it was"

    step "6. A YOLO rerun has nothing to apply"
    run_reconfig -y -c "$CONFIG"
    assert_exit 0
    assert_unchanged_run

    step "7. A staged desired state adds jq through mise"
    run_reconfig -y -c "$STRAY_CONFIG"
    assert_exit 0
    assert_output "✓ - Mise applied"
    mise config get --global "tools.$STRAY_TOOL" >/dev/null || fail "$STRAY_TOOL is not in mise's global config"
    echo "✓ $STRAY_TOOL is in mise's global config"

    step "8. Back to config.yaml, with an extraneous formula: a dry run shows the cleanup"
    brew install --formula "$EXTRANEOUS_FORMULA" >/dev/null 2>&1 || fail "brew install $EXTRANEOUS_FORMULA failed"
    run_reconfig -n -c "$CONFIG"
    assert_exit 0
    assert_output "✗ - $STRAY_TOOL is a stray in mise's global config"
    assert_output_line " mise uninstall $STRAY_TOOL@$STRAY_VERSION"
    assert_output_line " mise unuse --global --no-prune $STRAY_TOOL"
    assert_output_line " HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --formula $EXTRANEOUS_FORMULA"
    assert_none_applied

    step "9. YOLO cleans up, and a rerun has nothing to apply"
    run_reconfig -y -c "$CONFIG"
    assert_exit 0
    if mise config get --global "tools.$STRAY_TOOL" >/dev/null 2>&1; then
        fail "$STRAY_TOOL is still in mise's global config"
    fi
    if mise ls --installed --json "$STRAY_TOOL" | grep -q '"version"'; then
        fail "$STRAY_TOOL is still installed by mise"
    fi
    if brew list --formula "$EXTRANEOUS_FORMULA" >/dev/null 2>&1; then
        fail "$EXTRANEOUS_FORMULA is still installed"
    fi
    echo "✓ removed: the $STRAY_TOOL stray, its version, and $EXTRANEOUS_FORMULA"
    run_reconfig -y -c "$CONFIG"
    assert_exit 0
    assert_unchanged_run

    echo
    end_step
    echo "All e2e steps passed in $(seconds_since "$scenario_start")"
}

# run_reconfig runs reconfig with the given arguments and no input, saving
# its output in $OUT and its exit code in $CODE, then prints how long it
# took.
run_reconfig() {
    run_reconfig_with_input "" "$@"
}

# run_reconfig_with_input runs reconfig as run_reconfig does, piping the
# first argument to it as input.
run_reconfig_with_input() {
    local input=$1
    shift
    OUT=$(mktemp)
    echo "Running: reconfig $*"
    local start=$EPOCHREALTIME
    if [ -n "$input" ]; then
        echo "$input" | reconfig "$@" >"$OUT" 2>&1
        CODE=${PIPESTATUS[1]}
    else
        reconfig "$@" </dev/null >"$OUT" 2>&1
        CODE=$?
    fi
    local took
    took=$(seconds_since "$start")
    sed 's/^/  | /' "$OUT"
    echo "- exited after $took"
}

# snapshot describes what reconfig manages and a dry run must not change:
# the repository's core files, links in the home directory, and Bun's
# global completion.
snapshot() {
    find core -type f -exec sha256sum {} + | sort
    find ~ \( -path ~/.linuxbrew -o -path ~/.cache \) -prune -o -type l -printf '%p -> %l\n' | sort
    ls -l "$BUN_COMPLETION" 2>&1
}

# assert_unchanged checks that the snapshot is the given one.
assert_unchanged() {
    [ "$(snapshot)" = "$1" ] || fail "managed files changed: $(diff <(echo "$1") <(snapshot))"
    echo "✓ no managed file, link or completion changed"
}

# assert_none_applied checks that the run applied no plan.
assert_none_applied() {
    if grep -q '^✓ - .* applied$' "$OUT"; then fail "a plan was applied: $(grep '^✓ - .* applied$' "$OUT")"; fi
    echo "✓ nothing applied"
}

# assert_unchanged_run checks that every reconciler found nothing to do.
assert_unchanged_run() {
    assert_none_applied
    for line in "All formulae and casks are up to date" "No missing formulae" "No unwanted formulae" \
        "No missing mise versions" "No mise defaults to change" "No unwanted mise versions" \
        "No strays in mise's global config" "No missing or outdated npm globals" \
        "No unwanted npm globals" "No completion files to write" "No stow operations"; do
        assert_output_line "✓ - $line"
    done
}

# assert_bun_completion checks Bun's global completion: a file, not a link,
# with its regex bug fixed once.
assert_bun_completion() {
    [ -f "$BUN_COMPLETION" ] && [ ! -L "$BUN_COMPLETION" ] || fail "$BUN_COMPLETION is not a file"
    local fixes
    fixes=$(grep -o 'return; local re_prev_script=' "$BUN_COMPLETION" | wc -l)
    [ "$fixes" -eq 1 ] || fail "$BUN_COMPLETION has $fixes regex fixes, want 1"
    echo "✓ bun completion written, fixed once: $BUN_COMPLETION"
}

# assert_resolves checks that a link leads to the given file.
assert_resolves() {
    [ -L "$1" ] || [ -L "$(dirname "$1")" ] || fail "$1 is not linked"
    [ "$(readlink -f "$1")" = "$2" ] || fail "$1 resolves to $(readlink -f "$1"), want $2"
    echo "✓ $1 -> $2"
}

# seconds_since prints the time elapsed since the given $EPOCHREALTIME, as
# in 23.92s.
seconds_since() {
    awk -v s="$1" -v e="$EPOCHREALTIME" 'BEGIN { printf "%.2fs", e - s }'
}

# npm_global_installed reports whether $NPM_GLOBAL is a top-level global
# package. Not `npm ls -g`: that also finds the semver bundled inside npm.
npm_global_installed() {
    [ -d "$(npm root -g)/$NPM_GLOBAL" ]
}

# bun_installed reports whether mise has any bun version installed.
bun_installed() {
    mise ls --installed --json bun | grep -q '"version"'
}

# step ends the current step, then starts the next one.
step() {
    end_step
    echo
    echo "## $1"
    STEP_START=$EPOCHREALTIME
}

# end_step prints how long the current step took, if one has started.
end_step() {
    [ -n "${STEP_START:-}" ] || return 0
    echo "- step took $(seconds_since "$STEP_START")"
}

fail() {
    echo "✗ FAIL: $1"
    exit 1
}

assert_exit() {
    [ "$CODE" -eq "$1" ] || fail "reconfig exited $CODE, want $1"
    echo "✓ exit code $1"
}

assert_output() {
    grep -qF -- "$1" "$OUT" || fail "output lacks: $1"
    echo "✓ output has: $1"
}

# assert_section_output checks that the "## <section>" block of the output,
# up to the next "## " heading, contains the given text.
assert_section_output() {
    awk -v h="## $1" '$0 == h { on = 1; next } /^## / { on = 0 } on' "$OUT" | grep -qF -- "$2" ||
        fail "section $1 lacks: $2"
    echo "✓ section $1 has: $2"
}

# assert_output_line checks that the output has a line exactly equal to the
# given text.
assert_output_line() {
    grep -qxF -- "$1" "$OUT" || fail "output lacks line: $1"
    echo "✓ output has line: $1"
}

assert_equal() {
    [ "$2" = "$3" ] || fail "$1: got '$3', want '$2'"
    echo "✓ $1: $2"
}

# words_sorted prints its space-separated words sorted, on one line.
words_sorted() {
    tr ' ' '\n' <<<"$1" | grep -v '^$' | sort | xargs
}

main
