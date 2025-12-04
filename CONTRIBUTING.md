# Contributing to Dev Agent

Thanks for helping improve Dev Agent, a Go 1.21 CLI that automates test-driven workflows through specialist MCP agents. This document explains how to set up your environment, understand the codebase, and land high-quality contributions.

## Getting Started / Development Setup

### 1. Install prerequisites
- Go **1.21.x** (download from [go.dev](https://go.dev/doc/install) or your package manager).
- Git 2.40+ with an SSH or HTTPS credential manager.
- Optional but recommended: Docker (for running external services) and [`direnv`](https://direnv.net/) for managing environment variables.
- Tooling: `gofmt` ships with Go; install `golint` via `go install golang.org/x/lint/golint@latest` and (optionally) `golang.org/x/tools/cmd/goimports` for automatic import sorting.

### 2. Clone and bootstrap
```bash
git clone git@github.com:IANTHEREAL/dev_agent.git
cd dev_agent
# The Go module lives inside the nested dev_agent/ directory
cd dev_agent
go mod download
```

### 3. Configure environment
- Copy `.env.example` (if present) or create `.env` with the variables described in `SKILL.md` (usage notes) and `AGENTS.md` (agent runtime requirements). At minimum you will need `AZURE_OPENAI_API_KEY`, `MCP_BASE_URL`, and GitHub credentials for publishing branches.
- You can also export variables directly in your shell:
  ```bash
  export AZURE_OPENAI_API_KEY=...
  export MCP_BASE_URL=http://localhost:8000/mcp/sse
  export PROJECT_NAME=my-sample-project
  ```

### 4. Run the CLI locally
```bash
go run ./cmd/dev-agent --task "Smoke test" --parent-branch-id 00000000-0000-0000-0000-000000000000 --headless
```

If you are new to the architecture, read `AGENTS.md` for an end-to-end specification of the orchestrator and `SKILL.md` for the user-facing workflow before making major changes.

## Project Structure Overview
- `AGENTS.md` — system design for MCP-powered automation; reference this when modifying orchestration logic or publish flows.
- `SKILL.md` — concise how-to for running the CLI; keep it in sync whenever you change user-facing behavior.
- `docs/stream-json.md` — describes the streaming JSON protocol emitted by the CLI.
- `dev_agent/cmd/dev-agent/main.go` — CLI entry point; parses flags and bootstraps orchestration.
- `dev_agent/internal/brain` — wraps Azure OpenAI calls and prompt lifecycle.
- `dev_agent/internal/orchestrator` — coordinates Implement → Review → Fix loops and publish steps.
- `dev_agent/internal/tools` — Pantheon MCP client plus helper utilities.
- `dev_agent/internal/config`, `internal/logx`, `internal/streaming` — configuration loading, logging, and SSE helpers shared across the app.

Understanding these folders (and their documentation counterparts) will make code reviews smoother and reduces the chances of regressions.

## Code Style Guidelines
- **Formatting:** run `gofmt -w .` (or integrate it with your editor). Pull requests that are not gofmt-clean will be asked to reformat.
- **Linting:** run `golint ./...` and fix warnings or document why a warning is acceptable. You may also run `go vet ./...` for static analysis.
- **Imports & modules:** keep imports grouped (stdlib, third-party, internal) and prefer explicit module paths over relative ones. Use `go mod tidy` if you add/remove dependencies.
- **Errors & logging:** wrap errors with context (`fmt.Errorf("describe: %w", err)`) and emit actionable log messages through `internal/logx`.
- **Tests first:** follow the TDD loop the tool itself enforces—write or update a failing test before implementing behavior, whenever practical.

## Testing Guidelines
- Run fast unit tests before every commit:
  ```bash
  go test ./...
  ```
- When touching concurrent code or streaming logic (`internal/streaming`, `internal/tools`), also run race detection:
  ```bash
  go test -race ./...
  ```
- Aim for at least **80% coverage** in the packages you modify. You can inspect coverage via:
  ```bash
  go test ./... -coverprofile=coverage.out
  go tool cover -func=coverage.out
  ```
- Use table-driven tests and clearly named fixtures. If you add new top-level features, consider creating an integration-style test that exercises the CLI through `cmd/dev-agent`.
- Update `AGENTS.md`, `SKILL.md`, or `docs/stream-json.md` whenever new behaviors affect their contracts, and cross-link the relevant tests in your PR description.

## Pull Request Process
1. **Branching:** create a topic branch from `main` (`git checkout -b username/brief-topic`). Keep branches focused on a single issue.
2. **Commit messages:** follow `type: summary` conventions (e.g., `feat: add orchestrator retries`). Reference the GitHub issue number in either the commit body or PR description.
3. **Checks before push:**
   ```bash
   gofmt -w .
   golint ./...
   go test ./...
   ```
4. **Documentation:** update `AGENTS.md`, `SKILL.md`, or `docs/*.md` when the user experience or contracts change. Link to the relevant section in your PR.
5. **Reviews:** open a pull request that summarizes the change, highlights testing evidence, and calls out any follow-up TODOs. At least one maintainer review is required; address feedback with additional commits rather than force-pushing unless you are asked to rebase.
6. **Merging:** keep your branch up to date with `main` (prefer `git pull --rebase origin main`) to minimize conflicts. Maintainers will squash or merge commits depending on the change size.

## Testing Checklist Template
Include a short checklist in your PR description so reviewers can verify expectations quickly:
```
- [ ] gofmt
- [ ] golint ./...
- [ ] go test ./...
- [ ] Updated docs (AGENTS.md/SKILL.md/docs/stream-json.md as needed)
```

## Issue Reporting Guidelines
- **Bugs:** provide Dev Agent version (`git rev-parse HEAD`), Go version (`go version`), OS, reproducible steps, expected vs. actual behavior, and any log excerpts from `internal/logx`. Screenshots or JSON reports from failed runs are helpful.
- **Feature requests:** describe the workflow problem, why existing automation does not cover it, and which agent (implement vs. review) should own the new capability. Reference sections in `AGENTS.md` or `SKILL.md` if you believe they need to change.
- **Security disclosures:** do not open a public issue. Email the maintainers or use the private reporting channel described in the repository security policy (when available).
- Before filing, search existing issues and discussions to avoid duplicates. If you discover a related issue, add a comment with extra context instead of creating a new one.

## Need Help?
Open a discussion or ping maintainers on the issue you are exploring. Pointing to the relevant paragraphs in `AGENTS.md`, `SKILL.md`, or `docs/stream-json.md` accelerates responses and documents tribal knowledge for future contributors.
