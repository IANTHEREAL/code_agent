# Contributing to `dev_agent`

Thanks for your interest in improving the `dev_agent` TDD automation tool. This guide explains how to get a local environment running, the conventions we follow, and what we expect from pull requests and issues.

## Getting Started / Development Setup

1. **Install prerequisites**
   - [Go 1.21+](https://go.dev/dl/) with `$GOBIN` on your `PATH`.
   - Git 2.40+ and a GitHub account with SSH or HTTPS access.
   - Optional but helpful: `golangci-lint`, `staticcheck`, and `golang.org/x/lint/golint`.
2. **Clone and bootstrap**
   ```bash
   git clone git@github.com:IANTHEREAL/dev_agent.git
   cd dev_agent
   cd dev_agent          # inner directory is the Go module root
   go mod download
   ```
3. **Configure environment variables**  
   `AGENTS.md` lists every variable the CLI consumes (Azure OpenAI, Pantheon MCP, GitHub token, polling knobs, etc.). Create a `.env` file at the repository root or export the variables in your shell:
   ```bash
   # .env example (trim to what you need)
   AZURE_OPENAI_API_KEY=sk-...
   AZURE_OPENAI_BASE_URL=https://my-azure.openai.azure.com/
   AZURE_OPENAI_DEPLOYMENT=gpt-4o
   MCP_BASE_URL=http://localhost:8000/mcp/sse
   GITHUB_TOKEN=ghp_...
   ```
4. **Quick smoke test**
   ```bash
   cd dev_agent
   go run ./cmd/dev-agent --help
   ```
   If the CLI prints its usage banner, your toolchain is ready.

## Project Structure Overview

The repo root hosts contributor docs (`AGENTS.md`, `SKILL.md`, this file). The Go module lives in the nested `dev_agent/` directory. Highlights:

| Path | Description |
| --- | --- |
| `cmd/dev-agent/main.go` | CLI entry point that wires configuration, orchestrator, and final JSON output. |
| `internal/orchestrator` | Implements the implement → review → fix conversation loop described in `SKILL.md`. |
| `internal/brain` | Azure OpenAI client wrapper with retry/backoff logic. |
| `internal/tools` | Pantheon MCP helpers (`parallel_explore`, `read_artifact`, branch tracking). |
| `internal/config` | Environment / CLI parsing; mirrors the contract documented in `AGENTS.md`. |
| `internal/logx` | Lightweight structured logging helpers. |
| `internal/streaming` & `docs/stream-json.md` | NDJSON streaming event design and helpers. |

When in doubt about desired behaviors, read `AGENTS.md` for the high-level specification and `SKILL.md` for the expected TDD workflow and CLI usage.

## Code Style Guidelines

- **Formatting**: run `gofmt -w .` (or `gofumpt`) on touched files before committing. CI reviewers expect zero formatting diffs.
- **Imports**: keep imports grouped (`std`, `third-party`, `internal`). `goimports` can fix this automatically.
- **Linting**:
  ```bash
  golangci-lint run ./...
  go vet ./...
  golint ./...
  staticcheck ./...
  ```
  Run what you have installed; at minimum run `go vet ./...`.
- **Error handling**: prefer wrapped errors (`fmt.Errorf("...: %w", err)`) and avoid swallowing errors unless there is a logged justification.
- **Logging**: use `internal/logx` helpers for consistency and to keep streaming output machine-friendly.
- **Documentation**: add Go doc comments for exported structs/functions and update `AGENTS.md` / `SKILL.md` (or this file) when you evolve workflows.

## Testing Guidelines

The project is built around TDD (see `SKILL.md`), so every change should ship with tests.

```bash
cd dev_agent
go test ./...
go test ./... -race
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

- Keep package coverage from regressing; aim for ≥80% in any package you touch.
- Favor table-driven tests for orchestrator/tooling logic and use fakes for remote dependencies.
- When adding new features, describe your test plan in the PR body (what you ran, what you observed). Include sample snippets from `worklog.md` or other artifacts when it helps reviewers.

## Pull Request Process

1. **Branching**: branch from `main`. A typical pattern is `yourname/feature-short-description`. Example:
   ```bash
   git checkout -b jdoe/stream-json-failover
   ```
2. **Keep commits focused**: reference the related GitHub issue number in your commit messages (`docs: clarify publish steps for #52`).
3. **Before opening the PR**:
   - Run formatting, linting, and tests.
   - Verify `go test ./...` passes and that you updated docs if behavior changed.
   - Rebase on the latest `main` if your branch has been open for a while.
4. **PR checklist**:
   - Describe motivation, implementation, and test plan.
   - Link to the relevant issue (e.g., `Fixes #37`).
   - Attach screenshots or log excerpts when touching CLI UX or streaming output.
5. **Review etiquette**:
   - Expect at least one maintainer review.
   - Address review comments with additional commits (avoid force-pushing unless asked).
   - Clearly call out follow-up TODOs if something ships behind a feature flag.

## Issue Reporting Guidelines

- **Bug reports**: include
  - CLI command you ran (exact flags).
  - Environment details (OS, Go version, relevant env vars minus secrets).
  - Expected vs. actual behavior plus logs or excerpts from `worklog.md` / NDJSON stream.
  - Steps to reproduce (numbered list).
- **Feature requests**:
  - Describe the user story and why existing functionality is insufficient.
  - Reference sections of `AGENTS.md` / `SKILL.md` that would need updates.
  - Suggest acceptance criteria or success metrics.
- **Security disclosures**: do **not** file public issues; instead email the maintainers listed in the repository security policy (or use GitHub’s private reporting feature if enabled).

Providing concrete examples and context with every issue speeds up triage and ensures the MCP agents can replay the scenario during TDD iterations.

## Need Help?

- Start with `AGENTS.md` and `SKILL.md` for the conceptual model and CLI usage.
- Check `docs/stream-json.md` for streaming implementation details.
- If something is unclear, open a “question” issue or start a GitHub Discussion so maintainers can point you to the right place.

Welcome aboard, and thanks for helping make `dev_agent` better!
