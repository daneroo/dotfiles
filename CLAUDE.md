# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test/Lint Commands

- Go tests: `go test ./go/pkg/...` or `go test ./go/pkg/asdf/sort_test.go` for a single test
- Deno tests: `deno test ./deno/**/*_test.ts` or `deno test ./deno/reconfig_test.ts` for a single test
- Lint Go code: `go vet ./go/...` and `golangci-lint run`
- Lint Deno code: `deno lint ./deno/`
- Format Go code: `gofmt -w ./go/`
- Format Deno code: `deno fmt ./deno/`

## Code Style Guidelines

- **Markdown**: Use Markdownlint for linting.
- **Go**: Follow standard Go style with top-down approach (calling functions before helpers)
- **TypeScript**: Use Deno conventions, strict types, and modern TS features
- **Error Handling**: Use explicit error returns in Go, try/catch in TypeScript
- **Imports**: Group and sort imports (standard library first, then external, then local)
- **Naming**: Use camelCase for TS, camelCase for unexported Go, PascalCase for exported Go
- **Types**: Always define and use proper type definitions
- **Documentation**: Document all exported functions, types, and methods
- **Testing**: Always write tests for new functionality
- **Architecture**: Follow the reconciliation pattern: observe current state, establish desired state, determine and perform actions

Always ensure code is idempotent as this repository manages system dependencies and configurations.

## Claude Code Setup (2025-11-03)

We are currently using purchased credits, with API KEY as seen in [Anthropic's console](https://console.anthropic.com/settings/billing).

### Authentication Credentials

Claude Code stores its authentication credentials in the macOS Keychain:

- **Location**: Login Keychain (`~/Library/Keychains/login.keychain-db`)
- **Service Name**: "Claude Code"
- **Account**: Your username (e.g., "daniel")
- **Credential Type**: Anthropic API key (sk-ant-api03-...)

To view the stored credential:

```bash
security find-generic-password -s "Claude Code" -g 2>&1
```

Configuration files are stored in:

- `~/.claude.json` - Main configuration file
- `~/.claude.json.backup` - Backup configuration
- `~/.claude/` - Directory containing:
  - `debug/` - Debug logs
  - `projects/` - Project-specific data
  - `session-env/` - Session environment data
  - `shell-snapshots/` - Shell state snapshots
  - `statsig/` - Analytics data
  - `todos/` - Todo list data