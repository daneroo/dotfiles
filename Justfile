# Install just on Ubuntu:
# curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | sudo bash -s -- --to /usr/local/bin/
# Install just on macOS:
# brew install just

# List all available commands
default:
    @just --list

# Run all Go tests with verbose output
test:
    go test -v ./go/...

# Run tests with coverage
test-cover:
    go test -cover ./go/...
    # For HTML coverage report:
    # go test -coverprofile=coverage.out ./go/... && go tool cover -html=coverage.out
