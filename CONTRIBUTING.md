# Contributing to Dev Agent

## Welcome & Project Overview
Dev Agent is a Go 1.21 CLI that orchestrates a structured Test-Driven Development workflow by coordinating specialized MCP agents (`claude_code`, `review_code`) through Azure OpenAI Chat Completions. The CLI lives in this repository under the nested `dev_agent/` module and produces JSON reports describing the automated branch lineage, publish status, and review guidance. Review `AGENTS.md` for the full system specification before making changes.

## Getting Started
### Prerequisites
- Go 1.21 or newer
- Git and a GitHub account with access to your fork
- Access credentials for Azure OpenAI and MCP endpoints

### Clone the repository
```bash
git clone https://github.com/IANTHEREAL/dev_agent.git
cd dev_agent
```

### Repository layout
- `AGENTS.md`, `SKILL.md`, and top-level docs live at the repo root (`dev_agent/` once cloned).
- All Go sources, tests, and the CLI entrypoint live under the nested `dev_agent/` directory (from the workspace root this path is `dev_agent/dev_agent/`).
- Design notes (for example, the `--stream-json` emitter) live under `dev_agent/docs/` (workspace path `dev_agent/dev_agent/docs/`).

## Environment Setup
### Required environment variables
Set the following before building or running the CLI (placeholders shown):
```bash
export AZURE_OPENAI_API_KEY=sk-...
export AZURE_OPENAI_BASE_URL=https://your-resource.openai.azure.com
export AZURE_OPENAI_DEPLOYMENT=dev-agent
export GITHUB_TOKEN=ghp_...
export GIT_AUTHOR_NAME="Your Name"
export GIT_AUTHOR_EMAIL="you@example.com"
```
`AZURE_OPENAI_BASE_URL` must start with `https://`. The CLI also requires `--parent-branch-id` and `PROJECT_NAME` (via env or `--project-name`).

### Optional environment variables
- `AZURE_OPENAI_API_VERSION` (defaults to `2024-12-01-preview`)
- `MCP_BASE_URL` (defaults to `http://localhost:8000/mcp/sse`)
- `MCP_POLL_INITIAL_SECONDS`, `MCP_POLL_MAX_SECONDS`, `MCP_POLL_TIMEOUT_SECONDS`, `MCP_POLL_BACKOFF_FACTOR`
- `PROJECT_NAME` (used in reports) and `WORKSPACE_DIR` (defaults to `/home/pan/workspace`)

### Using a `.env` file
`internal/config.FromEnv` calls `loadDotenv(".env")`, so the CLI only loads a `.env` file that sits in the directory where you execute its commands. Because all build/run snippets below assume you work inside the Go module directory, place your `.env` next to that module’s `go.mod` (repo-relative `dev_agent/.env`, workspace-relative `dev_agent/dev_agent/.env`). If you run the CLI from another directory, set the variables manually or pass `--project-name` / flags explicitly. Add key/value pairs (one per line) to avoid exporting secrets repeatedly:
```
AZURE_OPENAI_API_KEY=...
PROJECT_NAME=my-repo
```
Existing shell variables always win, so `.env` is safe to commit to `.gitignore`.

### Configure Git credentials
Run `~/.setup-git.sh` once per environment to populate `git config` with `GIT_AUTHOR_NAME` / `GIT_AUTHOR_EMAIL` and (optionally) a credential helper wired to `GITHUB_TOKEN`:
```bash
~/.setup-git.sh
```

## Building & Testing
All Go commands run from the nested module directory:
```bash
cd dev_agent
go build ./...
go test ./...
```

### Running the CLI
Provide the required environment variables plus `--parent-branch-id`:
```bash
cd dev_agent
PROJECT_NAME=my-svc \
AZURE_OPENAI_API_KEY=... \
AZURE_OPENAI_BASE_URL=... \
AZURE_OPENAI_DEPLOYMENT=... \
GITHUB_TOKEN=... \
GIT_AUTHOR_NAME="Dev Agent" \
GIT_AUTHOR_EMAIL="dev-agent@example.com" \
go run ./cmd/dev-agent \
  --parent-branch-id 123e4567-e89b-12d3-a456-426614174000 \
  --task "Refine CONTRIBUTING.md for dev_agent"
```

Use `--project-name` if you prefer to override `PROJECT_NAME`:
```bash
go run ./cmd/dev-agent \
  --parent-branch-id <uuid> \
  --project-name dev-agent \
  --task "Ship NDJSON streaming support"
```

Enable NDJSON event streaming (documented in `dev_agent/docs/stream-json.md`, i.e., `dev_agent/dev_agent/docs/stream-json.md` from the workspace root) with:
```bash
go run ./cmd/dev-agent \
  --parent-branch-id <uuid> \
  --task "Investigate flaky tests" \
  --stream-json
```
`--stream-json` forces headless mode and emits orchestration events to stdout.

## Development Workflow
- **Branching**: create feature branches from `main` (for example, `git checkout -b feat/contrib-docs`) and keep them short-lived.
- **Tests first**: write or update Go tests to cover new behavior, especially inside `internal/` packages.
- **Code organization**: CLI wiring lives in `cmd/dev-agent`, reusable logic in `internal/<package>`, and design docs in `dev_agent/docs/` (workspace path `dev_agent/dev_agent/docs/`).
- **Local validation**: run `go test ./...` (and integration commands, if applicable) before committing.
- **Commit hygiene**: group related changes, write descriptive commit messages, and include issue numbers (e.g., `Resolves #37`) when relevant.

## Submitting Changes
1. **Fork & sync**: Fork `IANTHEREAL/dev_agent`, clone your fork, and add `upstream` pointing to the canonical repo. Rebase frequently to stay current with `main`.
2. **Push feature branch**: `git push origin feat/my-change`.
3. **Open a Pull Request**: target `main`, describe the motivation, summarize implementation details, and enumerate tests/verification steps. Link to the GitHub issue using `Fixes #<number>` or `Resolves #<number>`.
4. **Respond to review**: maintainers will review for correctness, test coverage, and alignment with `AGENTS.md`. Update the branch with follow-up commits instead of force-pushing unless asked.
5. **Final checks**: ensure CI (if configured) stays green and documentation is up to date before requesting merge.

## Code Style & Standards
- **Formatting**: run `gofmt -w` (or `go fmt ./...`) on touched files. `goimports` can help maintain import order.
- **Linting**: use `golangci-lint` or `golint ./...` locally if available to catch common issues.
- **Error handling**: prefer `%w` wrapping (`fmt.Errorf("fetch config: %w", err)`) and avoid dropping errors—log via `internal/logx` when side effects are required.
- **Logging**: route CLI/runtime logs through `logx` to keep consistent formatting and headless friendliness.
- **Docs & comments**: update `AGENTS.md`, `dev_agent/docs/*.md` (workspace `dev_agent/dev_agent/docs/*.md`), and function comments when behavior or configuration changes. Explain why non-obvious logic exists.

## Getting Help
- **Issues**: open a [GitHub issue](https://github.com/IANTHEREAL/dev_agent/issues) for bugs or feature requests. Tag maintainers and describe reproduction steps plus environment details.
- **Discussions / clarifications**: start a discussion or comment on the relevant issue/PR if you need guidance on MCP flows, Azure configuration, or workflow expectations.
- **Reference docs**: 
  - [AGENTS.md](AGENTS.md) – canonical Dev Agent specification
  - [SKILL.md](SKILL.md) – legacy TDD agent overview (contains historical Python references)
  - [`dev_agent/docs/stream-json.md`](dev_agent/docs/stream-json.md) – NDJSON streaming emitter design (workspace path `dev_agent/dev_agent/docs/stream-json.md`)

If you are blocked or unsure which document to update, ask in an issue before implementing large changes.
