# Contributing to dev_agent

Thanks for helping improve `dev_agent`! This guide explains how to set up a
development environment, how we review changes, and where to find additional
context. Please read `AGENTS.md` for a deep architectural overview of the
Pantheon workflow and `SKILL.md` for usage expectations before diving into
substantial changes.

## Getting Started / Development Setup

1. **Prerequisites**
   - Go 1.21 or newer with `GOBIN` on your `PATH`.
   - Git 2.40+ and a GitHub account with SSH or HTTPS access.
   - Optional: `golangci-lint` and `markdownlint` for local linting feedback.
2. **Clone and bootstrap**
   ```bash
   git clone https://github.com/IANTHEREAL/dev_agent.git
   cd dev_agent/dev_agent
   go mod download
   go build ./cmd/dev-agent    # quick sanity build
   ```
3. **Environment configuration**
   - Create a `.env` in the repository root (see the example flow in `SKILL.md`)
     or export the required variables described in `AGENTS.md` (`AZURE_OPENAI_*`,
     `MCP_BASE_URL`, `PROJECT_NAME`, etc.).
   - When developing against Pantheon, keep the JSON logs (`worklog.md`,
     `code_review.log`) under `/home/pan/workspace` to match the defaults.
4. **Dependency management**
   - Run `go mod tidy` after adding or removing imports.
   - Vendor dependencies only when absolutely necessary; otherwise rely on Go
     modules for reproducibility.

## Code Style Guidelines

- Format all Go files with `gofmt` (or `goimports`) before committing:
  `gofmt -w dev_agent/...`.
- Run `golangci-lint run ./...` (or `go vet ./...` and `golint ./...` if you
  prefer individual tools) to catch style and correctness issues early.
- Favor clear, defensive error handling: wrap errors with `fmt.Errorf` and
  `errors.Join` where context matters, and return early when possible.
- Keep package-level comments and exported identifiers documented per
  effective-go conventions.
- When touching orchestration logic, keep the Implement → Review → Fix contract
  from `AGENTS.md` intact and document non-obvious control flow with a short
  comment.

## Pull Request Process

1. **Branching** – use descriptive branches based on the workstream, e.g.
   `pantheon/feature-short-desc` or `pantheon/docs-contributing-abcdef`. Avoid
   committing directly to `main`.
2. **Commits** – prefer small, logically grouped commits. Follow the conventional
   prefix style (`feat:`, `fix:`, `docs:`) and reference the GitHub issue number
   when applicable, e.g. `docs: add CONTRIBUTING.md for #37`.
3. **Reviews** – open a PR against `main`, fill out the template, and link any
   Pantheon branch IDs from your local `worklog.md`. Request review from a
   maintainer and avoid force-pushing after review begins unless asked.
4. **Completeness** – update relevant docs (`AGENTS.md`, `SKILL.md`,
   `dev_agent/docs/stream-json.md`) when behavior changes, and describe testing
   evidence in the PR body.

## Testing Guidelines

- Run unit tests frequently: `cd dev_agent && go test ./...`.
- For modules with external integrations (streaming, tools, orchestrator),
  prefer table-driven tests and mock the MCP or Azure clients.
- Capture coverage before submitting: `go test ./... -coverprofile=coverage.out`
  and inspect with `go tool cover -html=coverage.out`.
- When adding new commands or modifying CLI flows in `cmd/dev-agent`, include at
  least one integration-style test under `internal/orchestrator` or a smoke test
  that exercises the new flag or code path.
- Document any manual verification (e.g., running `dev-agent --help`) in your PR
  description so reviewers can repeat it.

## Issue Reporting Guidelines

- **Bugs** – include reproduction steps, expected vs. actual behavior, CLI
  output, pertinent `worklog.md` snippets, and the Pantheon branch ID. Mention
  the environment variables set and whether you were in headless or interactive
  mode as described in `AGENTS.md`.
- **Feature requests** – explain the use case, reference which agent behavior or
  tool needs to change, and link to relevant sections in `SKILL.md` or other
  docs.
- **Security issues** – do **not** open a public issue; email the maintainers or
  use GitHub’s security advisories.
- Search existing issues before filing a new one to avoid duplicates and weigh
  in with 👍 reactions or extra context instead of opening a fresh ticket.

## Project Structure Overview

The repository root contains documentation (`AGENTS.md`, `SKILL.md`) plus the Go
module under `dev_agent/`. Key directories:

- `dev_agent/cmd/dev-agent`: CLI entry point that wires configs, orchestrator,
  and JSON output (see `AGENTS.md` for runtime flow).
- `dev_agent/internal/brain`: Azure OpenAI client (`LLMBrain`) and prompt
  handling utilities.
- `dev_agent/internal/config`: Loads `.env` / environment variables and enforces
  polling defaults.
- `dev_agent/internal/logx`: Lightweight logging primitives shared by CLI and
  orchestrator.
- `dev_agent/internal/orchestrator`: Implements the Implement → Review → Fix
  loop, Pantheon tool wiring, and publish/report logic.
- `dev_agent/internal/streaming`: Helpers for stream JSON handling (see
  `dev_agent/docs/stream-json.md`).
- `dev_agent/internal/tools`: MCP client (`parallel_explore`, branch tracking)
  and tool dispatch helpers.

Refer to the docs listed above when modifying these areas to stay consistent
with the agent workflow and external contracts.
