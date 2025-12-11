# Install just on Ubuntu:
# curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | sudo bash -s -- --to /usr/local/bin/
# Install just on macOS:
# brew install just

# List all available commands
default:
    @just --list

# Edit this directory in it's symbolic location (Antigravity bug) ~/dotfiles-edit
edit:
    @echo "Antigravity bug: use workspace symbolic link ~/.dotfiles-edit"
    @ls -l ~/dotfiles-edit
    agy ~/dotfiles-edit

# Run all tests (Go and Deno)
test: test-go test-deno

# Run Go tests
test-go:
    go test ./go/...

# Run Deno tests
test-deno:
    cd deno && deno task test

test-go-pretty:
    go test -v ./go/... | grep -E "(--- PASS|--- FAIL)"

# Run tests with coverage
# test-go-cover:
#     go test -cover ./go/...
#     # For HTML coverage report:
#     # go test -coverprofile=coverage.out ./go/... && go tool cover -html=coverage.out
