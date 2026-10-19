#!/usr/bin/env bash
# The e2e scenario, run inside a fresh Homebrew container (see run.sh).
# It reconciles against e2e/config.yaml and asserts output, exit codes,
# and the resulting machine state at every step.
set -uo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DIR/.." && pwd)"
cd "$REPO_ROOT"

CONFIG=e2e/config.yaml
DESIRED_FORMULAE="hello jq tree"
EXTRANEOUS_FORMULA=figlet
NPM_GLOBAL=semver

echo "Container: $(uname -sm), $(brew --version | head -1), bash ${BASH_VERSION}"

main() {
    local scenario_start=$EPOCHREALTIME

    step "1. A fresh machine reports exactly the desired formulae as missing"
    run_reconfig
    assert_exit 0
    assert_no_output "## ASDF"
    assert_section_output "NPM Globals Section" "✗ - npm is not installed, skipping"
    assert_output "## CLI Completions Section"
    local install_line
    install_line=$(grep '^ brew install --formula ' "$OUT" | tail -1)
    assert_equal "missing formulae" "$DESIRED_FORMULAE" "$(words_sorted "${install_line# brew install --formula }")"

    step "2. The printed brew install line works"
    run_printed "$install_line"
    for f in $DESIRED_FORMULAE; do
        brew list --formula "$f" >/dev/null 2>&1 || fail "$f is not installed"
    done
    echo "✓ installed: $DESIRED_FORMULAE"

    step "3. A rerun has nothing missing and nothing extraneous"
    run_reconfig
    assert_exit 0
    assert_output "✓ - No missing casks/formulae"
    assert_output "✓ - No extraneous casks/formulae"

    step "4. A formula outside the desired state is reported extraneous"
    brew install --formula "$EXTRANEOUS_FORMULA" >/dev/null 2>&1 || fail "brew install $EXTRANEOUS_FORMULA failed"
    run_reconfig
    assert_exit 0
    assert_output "✓ - No missing casks/formulae"
    assert_output " brew uninstall --formula $EXTRANEOUS_FORMULA"

    step "5. Once node is present, reconfig prints the npm install line, which works"
    brew install --formula node >/dev/null 2>&1 || fail "brew install node failed"
    run_reconfig
    assert_exit 0
    assert_output "✓ - npm is installed"
    assert_output "✗ - npm: $NPM_GLOBAL is missing"
    npm_global_installed && fail "reconfig installed $NPM_GLOBAL; it should only print the command"
    echo "✓ npm global not installed by reconfig: $NPM_GLOBAL"
    assert_output_line " npm install -g $NPM_GLOBAL"
    # The combined line may also update npm itself, when the npm that comes
    # with node is outdated.
    local npm_install_line
    npm_install_line=$(grep '^ npm install -g ' "$OUT" | tail -1)
    run_printed "$npm_install_line"
    npm_global_installed || fail "npm global $NPM_GLOBAL is not installed"
    echo "✓ npm global installed: $NPM_GLOBAL"
    echo "Rerunning: nothing to do"
    run_reconfig
    assert_exit 0
    assert_output "✓ - $NPM_GLOBAL"
    assert_output "✓ - No missing or outdated npm globals"
    assert_output "✓ - No unwanted npm globals"
    assert_no_output "Installing..."
    assert_no_output " npm install -g "

    echo
    end_step
    echo "All e2e steps passed in $(seconds_since "$scenario_start")"
}

# run_reconfig runs reconfig against the e2e desired state, saving its
# output in $OUT and its exit code in $CODE, then prints how long it took.
run_reconfig() {
    OUT=$(mktemp)
    local start=$EPOCHREALTIME
    reconfig -c "$CONFIG" >"$OUT" 2>&1
    CODE=$?
    local took
    took=$(seconds_since "$start")
    sed 's/^/  | /' "$OUT"
    echo "- exited after $took"
}

# run_printed runs a command line that reconfig printed, failing the scenario
# if it fails, then prints how long it took.
run_printed() {
    echo "Running:$1"
    local start=$EPOCHREALTIME
    eval "$1" || fail "printed line failed:$1"
    echo "- exited after $(seconds_since "$start")"
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

assert_no_output() {
    if grep -qF -- "$1" "$OUT"; then fail "output has: $1"; fi
    echo "✓ output lacks: $1"
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
