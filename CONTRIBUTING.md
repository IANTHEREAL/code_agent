# Contributing to Dev Agent

Thanks for your interest in improving Dev Agent! This project coordinates multiple MCP agents to run a disciplined TDD loop, so every contribution directly impacts the developer experience described in `AGENTS.md` and `SKILL.md`. The guidelines below explain how to set up your environment, code confidently, and land high-quality pull requests.

## Getting Started / Development Setup

### Prerequisites
- Go 1.21+ with `GOBIN` on your `PATH`.
- Git, a recent bash-compatible shell, and `make` (optional but helpful for scripting workflows).
- Access to the Pantheon MCP deployment and Azure OpenAI credentials if you plan to exercise the live orchestration paths (see `SKILL.md`).

### Clone and bootstrap
```bash
# 1. Clone
git clone git@github.com:IANTHEREAL/dev_agent.git
cd dev_agent

# 2. Inspect docs to understand the architecture
cat AGENTS.md
cat SKILL.md

# 3. Install Go dependencies
cd dev_agent
go mod download

# 4. Build / smoke-test the CLI
go build ./cmd/dev-agent
./dev-agent --help
```

### Environment configuration
- Copy environment variables from your own secrets manager into a local `.env` at the repo root if you intend to run the CLI end-to-end. The variables documented in `AGENTS.md` and `SKILL.md` (e.g., `AZURE_OPENAI_API_KEY`, `MCP_BASE_URL`, `PROJECT_NAME`) are read by `internal/config`.
- Use Go workspaces or `asdf`/`gvm` if you juggle multiple Go versions; Dev Agent targets Go 1.21.
- Run `go env GOPATH` to confirm that module caching works—`go mod tidy` should not report changes on a clean checkout.

## Code Style Guidelines
- **Formatting**: run `gofmt -w` (or `goimports`) on every touched Go file. CI assumes gofmt’d code and reviewers will request fixes otherwise.
- **Linting**: prefer [`golint`](https://pkg.go.dev/golang.org/x/lint/golint) or `golangci-lint` locally. At a minimum, run `go vet ./...` to catch obvious issues.
- **Naming**: follow standard Go conventions (exported identifiers in `internal` are rare; public APIs live under `cmd` and `internal/...` only when necessary).
- **Error handling**:
  - Return wrapped errors (`fmt.Errorf("context: %w", err)`) so orchestrator logs stay actionable.
  - Avoid panics outside of `main`. Prefer sentinel errors or typed errors when the orchestrator needs to branch on failure modes.
- **Logging**: route human-readable diagnostics through `internal/logx` to keep CLI output consistent. Reserve `fmt.Println` for structured JSON responses.
- **Documentation**: keep the Go doc comments for exported functions/classes accurate. Reference supporting material in `dev_agent/docs/stream-json.md` when touching streaming output logic.

## Pull Request Process
1. **Branch naming**: `yourname/topic-issue##` (e.g., `pantheon/fix-streaming-42`). Keep branches focused on a single change-set.
2. **Sync**: regularly rebase on `origin/main` to pick up orchestrator and tooling updates.
3. **Commits**: write imperative, descriptive messages such as `feat(orchestrator): add exponential backoff controls`. Squash only if it improves clarity.
4. **Checklist before pushing**:
   - `go test ./...`
   - `gofmt`/`golint`
   - Updated docs when behavior changes (`AGENTS.md`, `SKILL.md`, or this file).
5. **Pull request template**:
   - Reference the GitHub issue (e.g., "Fixes #37").
   - Summarize user-facing impact and testing performed.
   - Mention any follow-up tasks or TODOs explicitly.
6. **Reviews**: expect maintainers to focus on TDD workflow integrity, Pantheon MCP contracts, and compatibility with Azure OpenAI throttling described in `AGENTS.md`. Address review feedback promptly; follow-up commits should note "address review" in their body.

## Testing Guidelines
- **Unit tests**: add/update tests alongside code in `internal/...`. Run the whole suite with:
  ```bash
  cd dev_agent
  go test ./...
  ```
- **Integration tests**: flows that exercise the orchestrator + MCP streaming should live near the orchestrator or tools packages. Use build tags (e.g., `//go:build integration`) so they do not block quick feedback loops.
- **CLI smoke tests**: after substantial changes to `cmd/dev-agent`, run `go run ./cmd/dev-agent --task "echo" --parent-branch-id <uuid>` with mock env vars to confirm JSON output still matches the contract in `AGENTS.md`.
- **Coverage**: ensure new logic is covered. A simple baseline:
  ```bash
  go test ./... -coverprofile=coverage.out
  go tool cover -func=coverage.out
  ```
  Target ≥80% coverage within the packages you touch.
- **Determinism**: avoid relying on live Azure/OpenAI endpoints inside automated tests; inject interfaces and use fakes/mocks.

## Issue Reporting Guidelines
When filing an issue, please include:
- **Type**: `bug`, `feature`, or `docs`.
- **Description**: concise summary plus detailed behavior. If it relates to the TDD workflow, reference the relevant phase (Implement/Review/Fix) from `AGENTS.md`.
- **Reproduction steps**: commands executed, env vars, sample task input, and branch IDs observed.
- **Expected vs. actual**: highlight mismatched JSON fields, missing MCP events, or CLI regressions.
- **Logs & artifacts**: snippets from `internal/logx` output, `worklog.md`, or `dev_agent/docs/stream-json.md` traces when applicable.
- **Feature requests**: describe the user story and acceptance criteria, linking to companion documentation you expect to touch (e.g., `SKILL.md`).

## Testing Bug Template
```
### Summary
<one sentence>

### Environment
- OS / shell
- Go version
- MCP endpoint / Azure region

### Steps to Reproduce
1. ...
2. ...

### Expected
<describe>

### Actual
<describe>

### Additional Context
- Related issue or PR
- Logs, screenshots, or JSON output excerpts
```

## Project Structure Overview
- `dev_agent/cmd/dev-agent/main.go`: CLI entry point that wires configuration, orchestrator setup, and final JSON reporting (see "Overview" in `AGENTS.md`).
- `dev_agent/internal/brain`: wraps Azure OpenAI chat completions, retry logic, and prompt management.
- `dev_agent/internal/orchestrator`: enforces the Implement → Review → Fix discipline, manages agent conversations, and publishes branches.
- `dev_agent/internal/tools`: MCP client abstractions plus helpers for branch tracking.
- `dev_agent/internal/streaming`: utilities for streaming responses; refer to `dev_agent/docs/stream-json.md` when editing these flows.
- `dev_agent/internal/config`: centralizes env parsing and validation described under "Inputs and Configuration" in `AGENTS.md`.
- `AGENTS.md` & `SKILL.md`: high-level behavior and usage docs—update them whenever you add new user-visible features or CLI flags.

Keeping these guidelines in mind will help maintain Dev Agent’s reliability and keep the MCP-driven developer workflow smooth. Thanks again for contributing!
