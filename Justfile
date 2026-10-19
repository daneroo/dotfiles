# Install just on Ubuntu:
# curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | sudo bash -s -- --to /usr/local/bin/
# Install just on macOS:
# brew install just

# Show this helpful listing.
default:
    @just --list --unsorted

# Quality gate: run after edits and before commits.
ci: fmt-check vet test

# Reconcile this machine with config.yaml. Changes the machine (installs packages).
reconcile:
    ./check.sh

# Run Go tests.
test:
    go test ./go/...

# Run Go tests, showing only pass/fail lines.
test-pretty:
    go test -v ./go/... | grep -E "(--- PASS|--- FAIL)"

# Run end-to-end tests (in Docker).
test-e2e:
    ./e2e/run.sh

# Vet Go code.
vet:
    go vet ./go/...

# Format Go code in place.
fmt:
    gofmt -w ./go/

# Fail if any Go file needs formatting.
fmt-check:
    #!/usr/bin/env bash
    unformatted=$(gofmt -l ./go/)
    if [ -n "$unformatted" ]; then
        echo "Needs gofmt (run: just fmt):"
        echo "$unformatted"
        exit 1
    fi

# Run Deno tests (deno/ is slated for removal).
test-deno:
    cd deno && deno task test
