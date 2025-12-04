# Contributing to Dev Agent

Welcome! This document explains how to set up a development environment, run the Go toolchain, and contribute code or documentation to the `IANTHEREAL/dev_agent` project.

## Project Overview

Dev Agent is a Go 1.21 CLI (see `dev_agent/dev_agent/cmd/dev-agent`) that automates a disciplined **Implement → Review → Fix** workflow by orchestrating Azure OpenAI and Pantheon MCP agents (`claude_code` and `review_code`). The CLI loads configuration from environment variables (`internal/config`), keeps work artifacts such as `worklog.md`, and emits JSON summaries plus optional streaming telemetry (documented in `docs/stream-json.md`). See `AGENTS.md` for a deep dive into the architecture and runtime behavior.

## Prerequisites

### Tooling

- Go **1.21 or newer**
- Git + GitHub account (you will publish branches and open pull requests)
- Optional but helpful: `gh` CLI for inspecting issues/PRs and Pantheon MCP tooling for end-to-end tests

### Required Environment Variables

`internal/config/config.go` enforces the following variables at runtime:

| Variable | Purpose |
| --- | --- |
| `AZURE_OPENAI_API_KEY` | Azure OpenAI API key used by `internal/brain` |
| `AZURE_OPENAI_BASE_URL` | Azure endpoint (must start with `https://`) |
| `AZURE_OPENAI_DEPLOYMENT` | Deployment/model name passed to Chat Completions |
| `GITHUB_TOKEN` | Used when publishing branches from within the agent workflow |
| `GIT_AUTHOR_NAME` / `GIT_AUTHOR_EMAIL` | Commit identity applied during automated publish steps |

Additional knobs have sane defaults but can be overridden: `AZURE_OPENAI_API_VERSION`, `MCP_BASE_URL`, polling controls (`MCP_POLL_INITIAL_SECONDS`, `MCP_POLL_MAX_SECONDS`, `MCP_POLL_TIMEOUT_SECONDS`, `MCP_POLL_BACKOFF_FACTOR`), and workspace metadata (`PROJECT_NAME`, `WORKSPACE_DIR`). You can place these values in a `.env` file at the repo root; they are loaded before execution.

## Development Environment Setup

> **Nested module layout:** the Go module lives in `dev_agent/dev_agent`. The repository root also carries documentation such as `AGENTS.md` and `SKILL.md`.

1. **Fork & clone**
   ```bash
   git clone git@github.com:<your-user>/dev_agent.git
   cd dev_agent
   ```
2. **Enter the module root**
   ```bash
   cd dev_agent/dev_agent
   ```
3. **Configure environment**
   ```bash
   cat > .env <<'EOF'
   AZURE_OPENAI_API_KEY=...
   AZURE_OPENAI_BASE_URL=https://<your-endpoint>.openai.azure.com
   AZURE_OPENAI_DEPLOYMENT=<deployment-name>
   GITHUB_TOKEN=ghp_...
   GIT_AUTHOR_NAME="Your Name"
   GIT_AUTHOR_EMAIL=you@example.com
   EOF
   ```
   Add optional MCP and polling settings as needed.
4. **Install dependencies**: the project uses Go modules only; `go env GOPATH` is not required beyond a working Go toolchain.

## Building & Testing

Run these commands **from `dev_agent/dev_agent`**:

```bash
go build ./...
go test ./...
```

- `go build ./cmd/dev-agent` builds the CLI binary explicitly.
- `go test ./...` should pass cleanly before you open a PR.
- When validating feature work, also exercise `go run ./cmd/dev-agent --help` to ensure flags parse as expected.

## Code Contribution Workflow

1. **Sync with `main`**: `git fetch origin && git checkout main && git pull`.
2. **Create a topic branch**: `git checkout -b <feature-or-fix>` (reference the GitHub issue number when possible).
3. **Implement and test** using the TDD loop described in `AGENTS.md`.
4. **Keep `worklog.md` up to date** with the steps you took; the CLI expects it in `/home/pan/workspace/worklog.md`.
5. **Commit** with meaningful messages (imperative mood, mention issue numbers).
6. **Push** to your fork and open a Pull Request against `main`. Describe the change, validation steps, and any follow-up TODOs.

## Coding Standards & Conventions

- Follow canonical Go formatting (`gofmt` or `go fmt ./...`).
- Prefer small, self-contained functions and keep orchestration logic inside `internal/orchestrator`.
- Handle configuration/IO errors explicitly; bubble errors up instead of `panic`.
- Tests: add `go test` coverage for new logic. For CLI-only changes, include reasoning-based validation notes in the PR.
- Logging: use `internal/logx` rather than `fmt.Println` for runtime logs.

## Updating Documentation

- **Primary references**: keep `AGENTS.md` aligned with architecture or workflow changes, and update `docs/stream-json.md` whenever streaming events or flags change.
- **SKILL.md is outdated**: it still references the legacy Python CLI workflow. Leave it untouched unless you are specifically modernizing it; prefer linking to `AGENTS.md` and this CONTRIBUTING guide instead.
- When adding new features, document any user-facing behavior (flags, env vars, branch workflows) and mention how to test them.
- Include screenshots or JSON excerpts only when needed; prefer concise links back to the relevant files.

## Resources

- `AGENTS.md` — authoritative overview of agent roles, configuration, and orchestration loops.
- `docs/stream-json.md` — streaming telemetry design and event catalog.
- `internal/config/config.go` — source of truth for environment variables.
- Repository discussions/issues — use GitHub Issues (e.g., #37) to coordinate work before opening large PRs.
