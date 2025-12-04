# Contributing to dev_agent

Thanks for your interest in dev_agent — a Go CLI that coordinates Pantheon MCP
agents to practice disciplined TDD with Azure OpenAI. This guide explains how to
set up your environment, collaborate effectively, and land changes confidently.

## Getting Started

### Prerequisites

- Go 1.21 or newer (`go env GOVERSION` to confirm)
- Git + GitHub account with access to this repository
- Azure OpenAI deployment credentials and HTTPS endpoint URL
- Pantheon MCP endpoint reachable from your workstation
- Personal access token with `repo` scope exported as `GITHUB_TOKEN`

### Repository Setup

1. Fork the repo and clone your fork.

   ```bash
   git clone git@github.com:<you>/dev_agent.git
   cd dev_agent
   ```

2. The Go module lives under `dev_agent/`. Install dependencies and build.

   ```bash
   cd dev_agent
   go mod download
   go build ./cmd/dev-agent
   ```

3. Configure environment variables (or create a `.env` in the repo root).

   ```bash
   cat > .env <<'ENV'
   AZURE_OPENAI_API_KEY=...
   AZURE_OPENAI_BASE_URL=https://<your-azure-endpoint>
   AZURE_OPENAI_DEPLOYMENT=<model-deployment>
   GITHUB_TOKEN=ghp_xxx
   GIT_AUTHOR_NAME="Your Name"
   GIT_AUTHOR_EMAIL="you@example.com"
   PROJECT_NAME=demo-project
   MCP_BASE_URL=http://localhost:8000/mcp/sse
   ENV
   ```

   Optional knobs: `MCP_POLL_INITIAL_SECONDS`, `MCP_POLL_MAX_SECONDS`,
   `MCP_POLL_TIMEOUT_SECONDS`, and `MCP_POLL_BACKOFF_FACTOR`.

4. Sanity-check the CLI by running a task end to end.

   ```bash
   go run ./cmd/dev-agent \
     --parent-branch-id 123e4567-e89b-12d3-a456-426614174000 \
     --task "Smoke test dev_agent" \
     --project-name "$PROJECT_NAME" \
     --headless
   ```

   The command prints a JSON report on stderr; inspect it for obvious errors.

### Repository Tour

- `cmd/dev-agent/`: CLI entry point, flag parsing, and configuration wiring.
- `internal/config`: environment loading, validation, defaults, and worklog
  filenames.
- `internal/brain`: Azure OpenAI client with retry/backoff logic.
- `internal/orchestrator`: implement → review → fix loop, streaming hooks, and
  publish instructions.
- `internal/tools`: Pantheon MCP JSON-RPC client plus branch tracking handler.
- `docs/` and `AGENTS.md`: reference docs for streaming JSON and agent roles.

## Development Workflow

1. **Scout the issue** – Assign yourself (or coordinate in comments) and verify
   acceptance criteria, required inputs (branch IDs, workspace layout), and any
   missing telemetry.
2. **Cut a branch** – Branch from `main` using the shared
   `pantheon/<area>-<short-description>` style (for example,
   `pantheon/orchestrator-await-timeouts`). Avoid reusing branch names; the CLI
   surfaces them in publish reports.
3. **Maintain `worklog.md`** – Document analysis, design, validation notes, and
   follow-ups. Downstream agents rely on this file to understand workspace
   history.
4. **Follow the TDD loop** – Write (or update) tests that capture the issue,
   implement the fix, then rerun tests. Mirroring the Implement → Review → Fix
   cadence keeps automation predictable.
5. **Keep commits focused** – Group related code, tests, and docs together,
   write imperative commit messages, and reference the issue number.
6. **Update docs** – Touch `AGENTS.md`, `docs/stream-json.md`, CLI help, or
   other docs whenever behavior changes. New env vars must be documented under
   `internal/config` and user-facing docs.

## Code Style

- Run `gofmt`/`goimports` on every Go file (try `goimports -w .`).
- Prefer Go 1.21 features that aid readability but avoid experimental APIs.
- Use `internal/logx` instead of `fmt.Println` for runtime logging.
- Wrap errors with context using `fmt.Errorf("...: %w", err)` so failures are
  traceable.
- Keep package boundaries clean: orchestrator logic in `internal/orchestrator`,
  networking in `internal/tools`, configuration in `internal/config`.
- Exported identifiers need GoDoc comments that describe behavior and edge
  cases.
- For Markdown, follow ATX headings, fenced code blocks, and runnable samples.

## Testing

- Run the full suite with `go test ./...` from `dev_agent/` before every PR.
- Enable race detection for concurrency-heavy areas: `go test -race ./internal/...`.
- When changing orchestrator state machines, extend
  `internal/orchestrator/orchestrator_test.go` to capture instruction handling.
- Tooling code (`internal/tools`) should prefer fake MCP clients; avoid hitting
  real Pantheon endpoints in unit tests.
- If you adjust JSON streaming, add golden tests under `internal/streaming` and
  manually inspect `--stream-json` output via `jq`.
- Record any manual verification (live MCP runs, CLI transcripts) in
  `worklog.md` so reviewers can follow along.

## Pull Request Process

- Rebase your branch onto the latest `main` before opening a PR.
- Fill out the PR checklist:
  - [ ] Linked GitHub issue (for example, `Fixes #37`).
  - [ ] Summary describing what changed and why.
  - [ ] Validation notes (`go test ./...`, manual runs, linters).
  - [ ] Documentation updates included (CLI help, docs, worklog).
- CI currently runs `go test ./...`; keep it green locally to avoid churn.
- Address review comments promptly. Prefer follow-up commits over force-pushes
  unless a reviewer explicitly requests a cleanup rebase.
- After approval, a maintainer (or dev_agent itself) handles the publish step
  that runs `git commit`/`git push` inside the orchestrated workspace.

## Issue Guidelines

- **Bugs:** provide the exact CLI invocation (with secrets stripped), the JSON
  summary emitted by dev_agent, relevant snippets from `worklog.md` and
  `code_review.log`, and any Pantheon branch IDs involved. Outline repro steps,
  expected behavior, and the actual result.
- **Feature requests:** describe the user story, what automation outcome you
  expect, and any APIs or tools that must be added or updated.
- **Documentation improvements:** point to the file and section, describe the
  gap, and propose wording or examples you would like to see.
- Label issues appropriately (`bug`, `enhancement`, `docs`, etc.) and mention
  whether you plan to work on them yourself.
- Screenshots, logs, or sample JSON payloads are invaluable—attach them when
  possible.

We are excited to collaborate with you. If you run into setup snags or have
design questions, open a discussion or draft PR early so we can unblock you
quickly. Happy hacking!
