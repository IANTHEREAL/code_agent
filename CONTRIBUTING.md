# Contributing to dev_agent

Thanks for helping extend **dev_agent**, a Go 1.21 CLI that automates a disciplined Test-Driven Development (TDD) loop by orchestrating specialist MCP agents. This guide explains the expectations for working in this repository so your changes land smoothly and remain consistent with the existing automation model.

## Prerequisites
- **Tooling**: Go 1.21.x, Git 2.40+, and a POSIX-like shell. Install any editors or linters you prefer (VS Code, Goland, etc.).
- **Accounts & services**: Azure OpenAI access plus credentials for the Pantheon MCP endpoint and GitHub.
- **Environment variables** (loaded automatically from `.env` if present, then overridden by real env vars):
  | Variable | Required | Notes |
  | --- | --- | --- |
  | `AZURE_OPENAI_API_KEY` | Yes | Secret used by `internal/brain` for chat completions. |
  | `AZURE_OPENAI_BASE_URL` | Yes | Must begin with `https://` (see `internal/config`). |
  | `AZURE_OPENAI_DEPLOYMENT` | Yes | Deployment/model name. |
  | `AZURE_OPENAI_API_VERSION` | Optional | Defaults to `2024-12-01-preview`. |
  | `MCP_BASE_URL` | Optional | Defaults to `http://localhost:8000/mcp/sse`, must be HTTP(S). |
  | `MCP_POLL_INITIAL_SECONDS`, `MCP_POLL_MAX_SECONDS`, `MCP_POLL_TIMEOUT_SECONDS`, `MCP_POLL_BACKOFF_FACTOR` | Optional | Tune branch polling; constraints enforced in `internal/config`. |
  | `PROJECT_NAME` | Yes | Used in prompts and workspace artifact names. |
  | `WORKSPACE_DIR` | Optional | Defaults to `/home/pan/workspace`; determines where `worklog.md` and `code_review.log` are written. |
  | `GITHUB_TOKEN` | Yes | Needed for automated publish prompts. |
  | `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL` | Yes | Used when `dev_agent` commits/pushes branches. |

> Tip: keep secrets in a local `.env` file at the repo root; `internal/config` loads it first without overwriting existing environment variables.

## Development Environment Setup
1. Fork `IANTHEREAL/dev_agent` (or ensure you have push access) and clone your fork:
   ```bash
   git clone git@github.com:<you>/dev_agent.git
   cd dev_agent
   ```
2. The Go module lives in the nested `dev_agent/` directory. Enter it for any Go commands:
   ```bash
   cd dev_agent/dev_agent
   go env GOPATH   # optional sanity check
   ```
3. Create a `.env` with the variables listed above or export them in your shell.
4. Run the test suite once to ensure your environment is healthy:
   ```bash
   go test ./...
   ```
5. To exercise the CLI locally:
   ```bash
   go run ./cmd/dev-agent \
     --task "Add pagination to orders API" \
     --parent-branch-id 123e4567-e89b-12d3-a456-426614174000 \
     --project-name my-project
   ```
   Add `--stream-json` to mirror the NDJSON protocol described in `docs/stream-json.md`.

## Project Structure
```
repo root
├── AGENTS.md           # System overview & orchestrator responsibilities
├── SKILL.md            # High-level skill and CLI usage guidance
├── CONTRIBUTING.md     # This document
├── worklog.md          # Human-authored progress log (kept in workspace root)
└── dev_agent/
    ├── cmd/dev-agent   # CLI entrypoint (flags, config, orchestration kick-off)
    ├── internal/
    │   ├── brain       # Azure OpenAI client wrapper
    │   ├── config      # Environment loading & validation (see env table above)
    │   ├── orchestrator# Implements implement → review → fix loop & publish logic
    │   ├── tools       # MCP client + tool handler
    │   ├── streaming   # NDJSON event emitter used for --stream-json
    │   └── logx        # Minimal logging helper
    └── docs/           # Architecture notes, including `docs/stream-json.md`
```
Always run Go commands from `dev_agent/` because go.mod lives there.

## Running Tests
- Execute `go test ./...` from `dev_agent/` before sending a PR. The repository currently relies on the standard library test runner—no extra targets or makefiles exist.
- Add or update tests as you change behavior. The TDD automation expects red → green cycles, so match that discipline in manual contributions as well.

## TDD Automation Model
dev_agent enforces an **Implement → Review → Fix** cadence by delegating work to `claude_code` and `review_code` via the Pantheon MCP APIs:
1. **Implement**: `claude_code` applies code changes and records context in `worklog.md`.
2. **Review**: `review_code` audits the change and writes findings to `code_review.log`.
3. **Fix**: Additional implement iterations run until reviewers report zero P0/P1 issues or iterations are exhausted.
When contributing manually, mirror this mindset: write a failing test, implement the fix, request review, and loop until it is clean. Keep `worklog.md` up to date with the steps you performed so automated and human collaborators can replay your reasoning.

## Code Contribution Workflow
1. **Discuss**: Comment on the relevant GitHub issue (e.g., #37) with your intent before writing code.
2. **Branch**: From `main`, create a descriptively named feature branch, such as `pantheon/feat-<slug>` or `pantheon/fix-<bug>`. Keep one logical change per branch.
3. **Document Progress**: Update `worklog.md` (at the repo root/workspace root) during each Implement → Review → Fix loop. When using the automation, ensure `code_review.log` is present for reviewers.
4. **Implement with Tests**: Make incremental commits. Run `go fmt ./...`, `go test ./...`, and any additional checks you introduce.
5. **Commit**: Use clear messages (`feat: add NDJSON streaming doc`). Reference the issue number in the body when possible.
6. **Review**: Self-review your diff, ensure documentation (AGENTS.md, SKILL.md, docs/stream-json.md) stays in sync when behavior changes.
7. **Pull Request**: Push your branch and open a PR against `main`. Include:
   - Summary of the change and motivation
   - Test plan (`go test ./...` output or other tooling)
   - Links to updated docs or specs
   - Any follow-up TODOs
8. **Respond Quickly**: Address reviewer feedback, amend docs/tests, and keep the branch rebased if necessary.

## Code Style & Quality Expectations
- **Formatting**: Run `gofmt`/`goimports` on every Go file. Stick to standard Go naming conventions and module layout.
- **Error handling**: Prefer wrapped errors with context (`fmt.Errorf("context: %w", err)`). Avoid panics in production paths.
- **Logging**: Use `internal/logx` for structured logs; respect log levels selected by CLI flags.
- **Configuration**: Validate inputs via `internal/config`; do not duplicate env parsing elsewhere.
- **Docs**: Update AGENTS.md, SKILL.md, `docs/stream-json.md`, or new specs whenever behavior changes. Keep markdown ASCII-friendly and pass basic linting.
- **Tests**: Cover new logic with unit tests, especially around the orchestrator loop and streaming emitter. Mock Azure/MCP clients where practical.

## Issues, PRs, and Support
- **File issues** with clear titles, reproduction steps, expected vs. actual behavior, logs (scrub secrets), and environment details (Go version, OS).
- **Security/Sensitive topics**: Do not include secrets in issues. Email the maintainer listed on the GitHub profile if disclosure must be private.
- **Questions**: Use GitHub Discussions (if enabled) or open a “question” issue label.
- **PR readiness**: Make sure CI (or at least `go test ./...`) passes before requesting review. Reference your `worklog.md` entry so reviewers understand the context quickly.

## Additional References
- `AGENTS.md` – detailed dev_agent specification, configuration sources, and workflow guarantees.
- `SKILL.md` – quickstart instructions and high-level CLI usage tips.
- `docs/stream-json.md` – NDJSON streaming protocol for `--stream-json` mode.

Welcome aboard, and thank you for keeping dev_agent’s TDD workflow healthy!
