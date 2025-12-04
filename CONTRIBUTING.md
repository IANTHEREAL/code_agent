# Contributing to `dev-agent`

Thanks for helping improve the Dev Agent CLI. This project is a Go 1.21+ command-line tool that orchestrates an Implement → Review → Fix workflow via Azure OpenAI and Pantheon MCP agents (`claude_code` and `review_code`). The CLI lives under `/cmd/dev-agent`, reads configuration from `/internal/config`, and emits structured JSON summaries plus optional streaming NDJSON events (see `docs/stream-json.md`). Use this guide to set up your environment, follow the preferred workflow, and submit high-quality pull requests.

## Prerequisites

### Tooling
- Go 1.21 or newer (`go env GOVERSION` to verify).
- Git 2.40+ with working SSH/HTTPS access to GitHub.
- Access to Azure OpenAI resources and Pantheon MCP agents.
- A GitHub personal access token (PAT) with `repo` scope so `dev-agent` can publish branches.

### Required environment variables
`internal/config` enforces the following variables (load them via `export` or a `.env` file—`FromEnv` automatically reads `.env` in the repo root without overwriting existing shell values):

| Variable | Required? | Purpose |
| --- | --- | --- |
| `AZURE_OPENAI_API_KEY` | ✓ | Authenticates requests to Azure OpenAI Chat Completions. |
| `AZURE_OPENAI_BASE_URL` | ✓ | Azure endpoint (must start with `https://`). |
| `AZURE_OPENAI_DEPLOYMENT` | ✓ | Model/deployment name passed to Azure. |
| `AZURE_OPENAI_API_VERSION` | optional (defaults to `2024-12-01-preview`) | Version string for the Azure OpenAI API. |
| `MCP_BASE_URL` | optional (defaults to `http://localhost:8000/mcp/sse`) | Pantheon MCP SSE endpoint. Must be HTTP/HTTPS. |
| `MCP_POLL_INITIAL_SECONDS` | optional (default `2`) | Initial polling delay before MCP branch updates. |
| `MCP_POLL_MAX_SECONDS` | optional (default `30`) | Upper bound for polling backoff. |
| `MCP_POLL_TIMEOUT_SECONDS` | optional (default `600`) | Overall timeout while waiting for MCP branches. |
| `MCP_POLL_BACKOFF_FACTOR` | optional (default `2.0`) | Multiplier for exponential backoff (> 1.0). |
| `PROJECT_NAME` | optional | Human-readable project label included in reports. |
| `WORKSPACE_DIR` | optional (defaults to `/home/pan/workspace`) | Root path where agent worktrees/materialized files live. |
| `GITHUB_TOKEN` | ✓ | Used during publish prompts to push branches. |
| `GIT_AUTHOR_NAME` | ✓ | Commit author name applied by automation. |
| `GIT_AUTHOR_EMAIL` | ✓ | Commit author email applied by automation. |

> Tip: keep secrets out of shell history—populate `.env` and rely on `internal/config.loadDotenv` to import values.

## Repository layout and setup

This repository has a light-weight root plus a nested Go module:

```
dev_agent/         # Go module root (<repo>/dev_agent)
  cmd/dev-agent    # CLI entry point
  internal/...     # config, orchestrator, tools, streaming, etc.
AGENTS.md          # Architecture/system prompt reference
SKILL.md           # Historical Python CLI doc (outdated)
```

When cloning:

```bash
git clone git@github.com:IANTHEREAL/dev_agent.git
cd dev_agent              # repo root
touch .env                # create (or update) a local env file
# edit .env with the variables listed above, or export them in your shell
```

Always run Go commands from the module root:

```bash
cd dev_agent/dev_agent
go mod tidy     # optional, keeps dependencies clean
```

## Build, run, and test

All build/test operations must be executed from `dev_agent/dev_agent` (the Go module root). Common commands:

```bash
# Build the CLI (outputs ./dev-agent)
go build ./...

# Run unit/integration tests
go test ./...

# Execute the CLI in headless mode with a sample task
go run ./cmd/dev-agent --task "Demo task" --parent-branch-id <uuid>
```

Add any new tests alongside your code—`internal/orchestrator`, `internal/tools`, and the streaming utilities all expect unit coverage before reviewers accept changes.

## Development workflow

1. **Fork the repo** under your GitHub namespace, then clone your fork.
2. **Sync with `main`** before starting new work (`git fetch upstream && git rebase upstream/main`).
3. **Create a feature branch** from `main`. The automation uses `pantheon/<area>-<shortid>` (e.g., `pantheon/docs-9126bb`), and you should follow a similar, descriptive pattern that links back to the issue ID.
4. **TDD mindset**: implement code, run `go test ./...`, and iterate with the Implement → Review → Fix workflow described in `AGENTS.md`.
5. **Commit cleanly**: small, logically grouped commits with meaningful messages. Sign your commits if required by your org.
6. **Keep documentation in sync**: whenever you change agent behavior, update `AGENTS.md`, `docs/stream-json.md`, or other guides accordingly.
7. **Push and open a Pull Request** against `IANTHEREAL/dev_agent:main`. Reference the GitHub issue (e.g., “Fixes #37”) and summarize testing evidence.

## Coding standards & conventions

- **Formatting**: run `go fmt ./...` before committing. If you use `golangci-lint` or `gofumpt` locally, keep the output clean.
- **Imports**: standard library groups first, then external modules.
- **Errors**: wrap errors with context (`fmt.Errorf("context: %w", err)`), and log via `internal/logx`.
- **Logging**: keep logs structured and concise. Streaming mode (see `docs/stream-json.md`) should emit NDJSON events; do not print to stdout outside of orchestrator-approved locations.
- **Tests**: prefer table-driven tests. Derive new fixtures from actual MCP/Azure logs when possible.
- **Git hygiene**: never rewrite published history; avoid large binary blobs in commits.

## Documentation guidelines

- `AGENTS.md` is the canonical reference for the agent workflow/system prompt. Update it whenever behavior, required environment variables, or orchestration phases change.
- `docs/stream-json.md` governs the NDJSON streaming contract; expand or modify it whenever you introduce new event types or transport semantics.
- `SKILL.md` documents the **deprecated Python-era CLI** and is retained only for historical context. Do **not** copy its pip/virtualenv instructions into new docs—prefer the Go-based workflow described here.
- For any new subsystem, add a Markdown file under `dev_agent/docs/` and link it from this CONTRIBUTING guide or `AGENTS.md`.

## Pull request checklist

Before requesting review, verify:

- `go build ./...` and `go test ./...` succeed from `dev_agent/dev_agent`.
- All necessary environment variables are documented (and new ones validated in `internal/config`).
- Documentation updates accompany code changes when behavior shifts.
- Commits reference the GitHub issue number and describe the change + validation.
- You provided manual test notes (for documentation-only changes, explain the reasoning/validation you performed).

Welcome aboard, and thanks for keeping Dev Agent healthy!
