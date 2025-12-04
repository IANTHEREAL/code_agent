# Contributing to dev_agent

Thank you for your interest in improving **dev_agent**, the Go-based CLI that automates Pantheon task orchestration for MCP agents. This document explains how to set up your environment, follow the house style, and collaborate effectively with the maintainers.

## Code of Conduct
- Be respectful, inclusive, and patient in all interactions.
- Assume positive intent, give constructive feedback, and credit others' work.
- Report harassment or abusive behavior to the maintainers via GitHub issues or email listed in your commit signature.

We follow these simple guidelines instead of a formal policy. If you see behavior that violates them, please reach out.

## Getting Started
### Prerequisites
- Go 1.21+ and a working GOPATH.
- Git, access to GitHub, and (optionally) GitHub CLI for PR management.
- Tooling for linting and formatting (e.g., `golint`, `staticcheck`, `markdownlint`).

### Repository layout
- Top-level docs and specs live beside this file (`AGENTS.md`, `SKILL.md`).
- The Go module lives under `dev_agent/` with:
  - `cmd/dev-agent`: CLI entrypoint, argument parsing, and orchestration bootstrap.
  - `internal/`: configuration, orchestration loop, MCP tooling, and logging helpers.
  - `docs/`: feature-specific guides (e.g., streaming JSON behavior).

### Setup
```bash
# Clone and enter the repo
 git clone https://github.com/IANTHEREAL/dev_agent.git
 cd dev_agent

# Work inside the module directory when running Go commands
 cd dev_agent
 go mod tidy
```

## Development Workflow
1. Start from `main` and ensure it is up to date (`git pull origin main`).
2. Create a short-lived branch following the existing convention: `pantheon/<area>-<shortid>` (example: `pantheon/docs-73e73d8d`).
3. Make incremental commits that are easy to review. Prefer imperative commit messages ("add logging for..."), referencing issue numbers (e.g., `Resolves #37`).
4. Keep `/home/pan/workspace/worklog.md` for scratch notes; do not include it in commits.
5. Sync frequently with the remote repository and rebase when conflicts arise.

## Submitting Changes
- Push your feature branch and open a Pull Request against `main`.
- Clearly describe the problem, approach, and any trade-offs. Link to the GitHub issue ("Resolves #NN").
- Mention validation performed (tests, lint, manual verification) and provide reproduction steps if fixing a bug.
- Request at least one reviewer familiar with the touched area.

## Testing Guidelines
- Run tests from the module root (`cd dev_agent`).
  ```bash
  go test ./...
  ```
- Add unit tests for new functionality under the relevant package, especially inside `internal/`.
- For integration-style flows, exercise the CLI via `go test ./cmd/dev-agent` or an ad-hoc script to ensure arguments are parsed and orchestration succeeds.
- Use `go test -run TestName ./path` to focus on a scenario and `go test -race ./...` before merging when possible.

## Code Style and Formatting
- Run `gofmt -w` or `go fmt ./...` on all Go files before committing.
- `go vet ./...` and `golint ./...` (if installed) help catch mistakes early; please fix or document warnings.
- Keep packages cohesive: shared utilities belong in `internal/<area>`, while CLI-specific logic stays in `cmd/dev-agent`.
- Add doc comments for exported functions, types, and package-level variables.
- Prefer small, composable functions and clear error handling; log context with `internal/logx` helpers.

## Documentation Requirements
- Update `AGENTS.md`, `SKILL.md`, or `docs/` when behavior changes or new capabilities are introduced.
- When adding new commands, flags, or configuration fields, document them and include examples.
- Keep Markdown readable (80-100 character lines) and run `markdownlint` if available. Inline code blocks (for example, `go test ./...`) should specify language hints.

## Issue Reporting
- Use GitHub Issues for bugs and feature requests.
- Provide reproduction steps, logs (scrub sensitive tokens), and environment info (OS, Go version).
- For feature ideas, describe the motivation, proposed interface, and alternatives considered.
- Label issues appropriately so maintainers can triage quickly.

## Community and Communication
- Primary communication happens through GitHub Issues and Pull Requests.
- Draft PRs are welcome for early feedback—clearly mark TODOs.
- For urgent questions, mention the maintainer in the issue comments; asynchronous responses are the norm.

Your contributions keep dev_agent reliable for automated MCP workflows. Thank you for helping us build a disciplined, test-driven future!
