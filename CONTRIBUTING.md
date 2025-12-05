# Contributing to dev_agent

Thanks for helping improve the dev_agent CLI. This project automates a disciplined Implement → Review → Fix workflow by coordinating specialist MCP agents. Before you begin, skim the high-level behavior in [AGENTS.md](AGENTS.md) and the CLI usage notes in [SKILL.md](SKILL.md); implementation details for the event streamer live in [dev_agent/docs/stream-json.md](dev_agent/docs/stream-json.md).

## Getting Started
- **Prerequisites**: Go 1.21+, Git, and a working shell. The `gh` CLI is optional but useful for reviewing GitHub issues. Have access to the Azure OpenAI + Pantheon MCP credentials described in `AGENTS.md` (`AZURE_OPENAI_*`, `MCP_BASE_URL`, `PROJECT_NAME`, `WORKSPACE_DIR`, optional `GITHUB_TOKEN`).
- **Clone & layout**:
  ```bash
  git clone https://github.com/IANTHEREAL/dev_agent.git
  cd dev_agent/dev_agent
  ```
  The repo root contains docs (`AGENTS.md`, `SKILL.md`); the Go module lives in `dev_agent/` with `cmd/`, `internal/`, and `docs/` subdirectories.
- **Environment configuration**: create a `.env` file (loaded automatically) or export variables before running the CLI, e.g.
  ```bash
  cat > .env <<'EOF'
  PROJECT_NAME=my-project
  WORKSPACE_DIR=/home/pan/workspace
  AZURE_OPENAI_API_KEY=...
  AZURE_OPENAI_BASE_URL=...
  AZURE_OPENAI_DEPLOYMENT=dev-agent
  MCP_BASE_URL=http://localhost:8000/mcp/sse
  GITHUB_TOKEN=ghp_example  # optional, needed for publish
  EOF
  ```
- **Tooling**: install Go tooling (`gofmt`, `goimports`) and keep dependencies stdlib-only (module currently has no third-party deps). Run an initial build/test to confirm your setup:
  ```bash
  go build ./...
  go test ./...
  ```

## Development Workflow
- **Branching**: create topic branches from `main` named `pantheon/<scope>-<short-desc>` (example: `pantheon/docs-streaming-guide`). This mirrors the workflow used in Pantheon tasks (see the doc request branch naming instructions).
- **Worklog discipline**: maintain `/home/pan/workspace/worklog.md` throughout the task. Record context verification, design assumptions, validation checklists, and the final summary before opening a PR.
- **TDD mindset**: changes—docs included—should outline validation criteria before editing. For code, write or update tests before implementing behavior; for docs, capture the checklist you will verify (completeness, accuracy, link checks, etc.).
- **Commits**: keep them small and focused. Use present-tense summaries ("add streaming doc checklist") and reference the relevant GitHub issue (e.g., `Fixes #37`).
- **Formatting & tooling**: run `gofmt ./...` (or `go fmt ./...`) on any modified Go files and ensure `git diff` only shows intentional ASCII changes.

## Code Standards
- Follow idiomatic Go style (short identifiers scoped locally, prefer zero-values, avoid global state). Keep packages narrowly scoped inside `internal/{brain,config,logx,orchestrator,streaming,tools}` and add new packages sparingly.
- Place binaries under `cmd/<name>/main.go`; the primary entry point is `cmd/dev-agent/main.go`. Keep CLI flag plumbing simple and favor pure functions in `internal/` packages.
- Comments should explain intent, not mechanics, and only when the code is non-obvious (see `internal/orchestrator` for examples). Use ASCII characters unless the surrounding file already uses Unicode.
- When touching streaming code or tool integrations, consult [dev_agent/docs/stream-json.md](dev_agent/docs/stream-json.md) to keep event names and payload fields consistent.

## Testing Expectations
- Always run:
  ```bash
  go build ./...
  go test ./...
  ```
  before committing. Add focused tests in the nearest package (e.g., `go test ./internal/orchestrator`) when fixing bugs or adding features.
- For regressions, include a test that fails without your fix and passes with it; link the reproducer in the PR body.
- If you introduce new CLI flags or workflows, exercise the binary locally (e.g., `go run ./cmd/dev-agent --task "Smoke" --parent-branch-id <id>`) to make sure the UX described in SKILL/AGENTS remains accurate.

## Pull Request Process
1. Open or reference a GitHub issue describing the change (issue numbers are required in PR descriptions).
2. Create your branch, implement the change, and keep `worklog.md` up to date with design/validation notes.
3. Run `go build ./...` and `go test ./...` (plus any targeted suites) and capture the commands + results in the PR body.
4. Summarize key updates (docs touched, new flags, etc.), note any assumptions (such as missing LICENSE), and link to related Pantheon tasks if applicable.
5. Request review once validation is green. Expect reviewers to focus on TDD discipline and clarity; be ready to explain how your change honors the Implement → Review → Fix workflow.

## Communication
- Use GitHub Issues for bug reports or feature requests and Discussions (if enabled) for open questions.
- For real-time collaboration, comment on the relevant issue/PR with context, links to `worklog.md`, and Pantheon task IDs so maintainers can trace the lineage.
- If something blocks progress (missing config, unclear scope), ask in the issue before proceeding—capturing the answer in the worklog keeps the next contributor unblocked.

## License
This repository currently does **not** include a LICENSE file. By contributing you agree that your submissions may be used under the repository owner’s terms. If you require a formal license grant (MIT, Apache 2.0, etc.), open an issue to clarify the policy before submitting substantial work and mention any assumptions in your PR description.
