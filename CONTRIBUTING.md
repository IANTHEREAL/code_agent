# Contributing to dev_agent

## Welcome & Introduction
Thanks for your interest in dev_agent! This repository hosts a Go-based CLI that automates a disciplined Test-Driven Development (TDD) workflow by delegating work to Pantheon MCP agents (`claude_code` for implementation and `review_code` for audits). Before contributing, skim `AGENTS.md` for the orchestration overview and `SKILL.md` for the high-level usage contract so your changes align with the expected agent behaviour.

## Prerequisites
- Go **1.21+** (the module is pinned to 1.21 in `dev_agent/go.mod`).
- Git with access to GitHub.
- Optional: a `.env` file or exported environment variables (see `SKILL.md`) when you need to run the CLI against live services.
- A working installation of `rg (ripgrep)` is helpful for the validation script, but the script gracefully falls back to `grep`.

## Development Setup
1. **Clone the repo**
   ```bash
   git clone git@github.com:IANTHEREAL/dev_agent.git
   cd dev_agent
   ```
2. **Inspect the workspace** – root docs live at the repository root while the Go module sources sit under `dev_agent/`.
3. **Install Go toolchain dependencies** – no third-party modules are required beyond the Go standard library.
4. **Build the CLI** (from the module directory):
   ```bash
   cd dev_agent
   go build ./...
   ```
5. **Run the test suite**:
   ```bash
   go test ./...
   ```
6. **Optional runtime config** – create a `.env` (see `SKILL.md`) with Azure OpenAI and Pantheon MCP credentials before exercising the CLI (`cmd/dev-agent/main.go`).

## Project Structure Overview
- `AGENTS.md` – technical spec for the orchestration workflow and agent responsibilities.
- `SKILL.md` – quick-start skill description and CLI usage notes.
- `scripts/validate_contributing.sh` – lightweight documentation validator used in this change.
- `dev_agent/`
  - `cmd/dev-agent/` – CLI entry point and flag parsing layer.
  - `internal/brain/` – wraps Azure OpenAI completions.
  - `internal/config/` – environment + CLI configuration loading and validation.
  - `internal/logx/` – logging helpers (structured stdout/stderr).
  - `internal/orchestrator/` – core TDD loop, execute_agent prompts, publish flow.
  - `internal/streaming/` – streaming JSON emitter infrastructure (`docs/stream-json.md` describes the design).
  - `internal/tools/` – MCP client, tool handlers, branch-tracking utilities.
  - `docs/` – supplemental technical designs (e.g., `stream-json.md`).

## Development Workflow
1. **Create a feature branch** from `main`, following the Pantheon-style pattern `pantheon/<issue>-<short-id>` (e.g., `pantheon/issue-37-ab12cd34`).
2. **Record intent** in `worklog.md` at the workspace root: capture analysis, design, implementation decisions, validation results, and follow-ups to keep parity with AGENTS’ expectations.
3. **Follow TDD** – implement changes in small slices, cover them with targeted Go unit tests, run `review_code` (or local equivalent reviews) before you finalize.
4. **Keep the branch lineage clean** – each execute-agent style iteration should leave the workspace buildable and tested so publishing can occur as soon as `review_code` reports zero P0/P1 issues.
5. **Sync with main regularly** to reduce merge conflicts and to keep branch lineage metadata accurate for publish reports.

## Code Style & Best Practices
- Always run `gofmt`/`goimports` on Go files; keep code idiomatic and lint-clean (spot-check with `go vet ./...` when touching core packages).
- Prefer dependency injection over package-level globals; pass context/state explicitly through orchestrator layers.
- Use `internal/logx` for logging; keep logs structured and concise to preserve headless/streaming compatibility.
- Fail fast with descriptive errors rather than panics; wrap errors with context when crossing package boundaries.
- Keep changes focused: avoid mixing documentation updates, feature work, and refactors in a single PR unless necessary.
- Document tricky behaviour with succinct comments, especially in orchestrator/tooling logic that mirrors agent prompts.

## Testing Guidelines
- Run `go test ./...` from `dev_agent/` before every commit; use table-driven tests for orchestrator/config logic.
- When adding new functionality, add or update package-specific tests (e.g., `internal/orchestrator`, `internal/tools`) so `review_code` has evidence that behaviour is covered.
- Use `go test ./path/to/pkg -run TestName -count=1` for focused debugging when needed.
- Execute `go build ./...` to ensure the CLI compiles across all packages.
- Validate documentation requirements with `./scripts/validate_contributing.sh` so CI/reviewers can quickly verify required sections exist.

## Submitting Changes
1. **Self-checks** – run `./scripts/validate_contributing.sh`, `go fmt`, `go vet ./...`, `go build ./...`, and `go test ./...`.
2. **Commit messages** – follow the conventional `<type>: <summary>` format and reference the GitHub issue (e.g., `feat: add streaming emitter (#37)`). Multi-line messages can capture validation details and tooling provenance, mirroring the example in the issue instructions.
3. **Update documentation** – if you touch agent workflows or CLI behaviour, adjust `AGENTS.md`, `SKILL.md`, and any relevant docs.
4. **Open a PR** against `main`, link the GitHub issue, and describe validation plus branch lineage information (as surfaced in the publish report). Wait for automated review signals (including `review_code`) to report zero P0/P1 issues before requesting human review.

## Communication & Questions
- Use the GitHub Issue tracker for bugs or feature requests; tag maintainers if a design discussion is needed.
- Start a GitHub Discussion or comment on the relevant issue when you need clarification on MCP workflows, branch lineage requirements, or deployment details.
- For urgent questions about the publish workflow or agent orchestration, reference the `publish_report` emitted in the CLI output and follow up via GitHub discussions/comments with those details so maintainers can reproduce the context.
