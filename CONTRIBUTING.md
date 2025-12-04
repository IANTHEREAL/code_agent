# Contributing to dev_agent

Thanks for your interest in improving dev_agent! This document explains how to set up a development environment, follow our workflow, and submit high-quality contributions that keep the project healthy.

## Getting Started

### Prerequisites
- Go 1.21 or newer with `$GOPATH/bin` on your `PATH`
- Git 2.40+ for managing branches and remotes
- A GitHub account with a fork of `IANTHEREAL/dev_agent`
- Optional: a recent version of `golangci-lint` if you prefer richer local linting

### Local Setup
1. Fork the repository on GitHub and clone your fork:
   ```bash
   git clone https://github.com/<you>/dev_agent.git
   cd dev_agent
   ```
2. (Recommended) Point an `upstream` remote at the canonical repo so you can pull in updates:
   ```bash
   git remote add upstream https://github.com/IANTHEREAL/dev_agent.git
   ```
3. Install Go dependencies from the module root (`./dev_agent`):
   ```bash
   cd dev_agent/dev_agent
   go mod download
   ```
4. Return to the repository root when editing docs or managing git operations.

## Development Workflow
1. Sync `main` with upstream:
   ```bash
   git checkout main
   git pull upstream main
   ```
2. Create a feature branch. Use a descriptive name such as `feature/agent-events`, `fix/stream-race`, or the issue-oriented style `pantheon/issue-37-abc123`.
3. Make focused changes with clear, imperative commit messages (e.g., `Add stream JSON docs link`). Reference the GitHub issue number in the body when applicable.
4. Keep branches small and rebased on top of the latest `main` to simplify review. Resolve merge conflicts locally before opening the pull request.

## Code Standards
- Follow idiomatic Go style. Always run `gofmt` (or `goimports`) on touched files before committing.
- Prefer clear, descriptive names. Use mixedCaps for exported identifiers and avoid abbreviations unless they are widely understood (`ctx`, `cfg`, `id`).
- Keep functions small and focused. When adding new orchestration logic, consider whether it belongs in `internal/orchestrator` or in a dedicated package.
- Document exported types and any public-facing behaviors that may be consumed by other packages or tools.
- Run static checks where practical (`go vet ./...` or your preferred linter). Address warnings or explain them in the pull request.

## Testing
- Run the full test suite from the module root before pushing:
  ```bash
  cd dev_agent/dev_agent
  go test ./...
  ```
- For packages that interact with streaming or async behavior, seed deterministic randomness where possible and use `t.Parallel()` judiciously.
- Write table-driven tests for logic-heavy functions (e.g., planners inside `internal/brain` or adapters under `internal/tools`).
- When adding new features, accompany them with regression tests. If tests are hard to write, describe the limitation in the PR so reviewers understand the trade-offs.

## Pull Request Process
- [ ] Ensure your branch is up to date with `main` and rebased when necessary.
- [ ] Run `gofmt`, `go vet`, and `go test ./...` from `dev_agent/dev_agent`.
- [ ] Update or add documentation (`AGENTS.md`, `SKILL.md`, `docs/stream-json.md`, or this file) whenever behavior or public APIs change.
- [ ] Squash obvious fixup commits before opening the PR; keep logical commits separated when they aid review.
- [ ] Fill out the PR template, link the relevant issue, and clearly describe the motivation and testing performed.
- [ ] Be responsive to reviewer feedback—push follow-up commits rather than force-pushing unless a reviewer requests a clean history.

## Issue Reporting Guidelines
- Search existing issues before filing a new one to avoid duplicates.
- Clearly describe the problem, expected behavior, and actual behavior. Include stack traces, logs, or screenshots when helpful.
- Provide reproduction steps. For runtime issues, note your OS, Go version, and any environment configuration (`GOOS`, `GOARCH`, proxies`).
- For feature requests, explain the use case, the impact on current workflows, and any constraints or alternatives considered.
- Tag the issue appropriately (`bug`, `enhancement`, `docs`, etc.) so it can be triaged quickly.

## Project Structure Overview
The Go module lives under `./dev_agent` while SDK-style docs stay at the repository root. Key locations:

```
dev_agent/
├── cmd/dev-agent/          # CLI entrypoint and main process wiring
├── internal/
│   ├── brain/              # Planning, reasoning, and strategy code
│   ├── config/             # Configuration loading and validation
│   ├── logx/               # Logging helpers and adapters
│   ├── orchestrator/       # High-level orchestration and task flow
│   ├── streaming/          # Stream interfaces and helpers
│   └── tools/              # Tooling and integrations exposed to agents
├── docs/stream-json.md     # Reference for streaming JSON payloads
├── go.mod / go.sum         # Module definition pinned to Go 1.21
AGENTS.md                   # Agent concepts and usage guidelines
SKILL.md                    # Skill development and registration guide
CONTRIBUTING.md             # (this file) collaboration guide
```

If you are unsure where new code should live, open an issue or draft PR and ask—the maintainers are happy to help you find the right home.
