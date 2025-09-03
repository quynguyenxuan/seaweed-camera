# CRUSH.md for SeaweedFS

This file provides essential commands and code style guidelines for agentic coding agents working in this repository.

## Build Commands

*   **Default Build**: `cd weed; go install`
*   **Full Build (with all tags)**: `cd weed; go install -tags "elastic gocdk sqlite ydb tarantool tikv rclone"`

## Test Commands

*   **Run All Tests**: `cd weed; go test -tags "elastic gocdk sqlite ydb tarantool tikv rclone" -v ./...`
*   **Run a Single Test**: `cd weed; go test -v -run "TestName" ./path/to/package`
    *   Example: `cd weed; go test -v -run "TestMasterClient" ./wdclient`

## Linting & Formatting

*   **Format Code**: `gofmt -w .` (run from the `weed` directory for Go files)
*   **Static Analysis (Linting)**: `go vet ./...` (run from the `weed` directory)
*   **Recommended Linter (if installed)**: `golangci-lint run ./...` (run from the `weed` directory)

## Code Style Guidelines (Go)

*   **Imports**: Group imports into standard library, external, and internal blocks, separated by blank lines. `gofmt` handles most of this automatically.
*   **Formatting**: Adhere to `gofmt` standards.
*   **Naming Conventions**:
    *   Package names: short, all lowercase, no underscores.
    *   Variables/Functions: `camelCase` for unexported, `PascalCase` for exported.
    *   Acronyms: `HTTP`, `URL`, `ID` (e.g., `serveHTTP`).
*   **Error Handling**: Return errors explicitly as the last return value. Check errors immediately. Do not ignore errors.
*   **Types**: Use specific types over `interface{}` where possible.
*   **Comments**: Comment exported functions/methods and complex logic. Explain _why_, not just _what_.
