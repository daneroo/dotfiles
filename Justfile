# Install just on Ubuntu:
# curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | sudo bash -s -- --to /usr/local/bin/
# Install just on macOS:
# brew install just

# Recipes receive their arguments as "$@", so arguments with spaces stay whole.
set positional-arguments

# Show this helpful listing.
default:
    @just --list --unsorted

# Quality gate (gofmt check, vet, tests): run after edits and before commits.
ci: _fmt-check _vet _test

# Format Go code in place.
fmt:
    gofmt -w ./go/

# Run the end-to-end scenario in a fresh Homebrew container (Docker).
e2e:
    ./e2e/run.sh

# Reconcile this machine with config.yaml, asking before each plan (options pass through: just reconcile -n).
reconcile *args:
    ./reconfig.sh "$@"

[private]
_fmt-check:
    #!/usr/bin/env bash
    echo "== formatting: gofmt -l ./go/ =="
    unformatted=$(gofmt -l ./go/)
    if [ -n "$unformatted" ]; then
        echo "Needs gofmt (run: just fmt):"
        echo "$unformatted"
        exit 1
    fi

[private]
_vet:
    @echo "== vet: go vet ./go/... =="
    go vet ./go/...

[private]
_test:
    @echo "== tests: go test ./go/... =="
    go test ./go/...
