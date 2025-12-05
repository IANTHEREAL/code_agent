# Contributing to dev_agent

## Welcome & Introduction
`dev_agent` automates a disciplined implement → review → fix workflow by orchestrating specialist MCP agents through a Go CLI. We appreciate every issue, doc fix, and feature that helps keep the automation reliable and easy to operate. This document explains how to get set up, make changes confidently, and land them through a predictable review process.

## Getting Started
- **Prerequisites**
  - Go 1.21 or newer and Git.
  - Access to Azure OpenAI (for `AZURE_OPENAI_*` variables) and a Pantheon MCP endpoint if you plan to exercise the full workflow locally.
  - POSIX shell utilities (`bash`, `make`, `openssl`) available on most Linux/macOS environments.
- **Repository layout**
  - Top-level docs (`AGENTS.md`, `SKILL.md`, `CONTRIBUTING.md`).
  - Go module lives in `dev_agent/` (run build/test commands from this directory).
  - CLI entry point: `dev_agent/cmd/dev-agent/main.go`; internal packages under `dev_agent/internal/`.
- **Clone & bootstrap**
  1. `git clone https://github.com/IANTHEREAL/dev_agent.git`
  2. `cd dev_agent/dev_agent`
  3. `go env -w GO111MODULE=on` (optional on Go ≥1.21 but harmless)
  4. Install any tooling you rely on (e.g., `go install golang.org/x/tools/cmd/goimports@latest`).
- **Environment configuration** (mirrors `AGENTS.md` / `SKILL.md`):
  1. Copy `.env.example` if present, or create `.env` with at least `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_BASE_URL`, `AZURE_OPENAI_DEPLOYMENT`, `MCP_BASE_URL`, `PROJECT_NAME`, the GitHub token used for publish prompts, plus the Git author identity variables `GIT_AUTHOR_NAME` and `GIT_AUTHOR_EMAIL`.
  2. Set `GIT_AUTHOR_NAME` (e.g., `Jane Developer`) and `GIT_AUTHOR_EMAIL` (e.g., `jane@example.com`) to the values you want recorded in generated commits—`internal/config.FromEnv` refuses to run without them.
  3. Run `source .env` (or export variables directly) before invoking the CLI.
  4. Use `WORKSPACE_DIR=/home/pan/workspace` to keep logs (`worklog.md`, `code_review.log`) consistent across agents.

## Development Workflow
1. **Start with an issue** – document intent, edge cases, and acceptance criteria in GitHub before touching code.
2. **Create a feature branch** – `pantheon/<type>-<short-description>` keeps history readable (examples: `pantheon/feat-stream-json`, `pantheon/docs-contributing`).
3. **Update `/home/pan/workspace/worklog.md`** as you move through Phase 0 (context), Phase 1 (analysis/design), and Phase 2 (implementation/test). This mirrors the automated agents’ expectations and gives reviewers traceability.
4. **Follow TDD** – implement, review, and fix loops should be small. Prefer adding or updating tests before non-trivial code edits.
5. **Keep commits focused** – each commit should compile, include necessary docs, and reference the driving issue (e.g., “docs: add streaming JSON guide (fixes #12)”).
6. **Run formatters and sanity checks** before pushing (`go fmt`, `go test`, etc.).

## Code Standards
- **Formatting**: run `go fmt ./...` (or `gofmt -w` on touched files). Keep imports grouped (`goimports`).
- **Structure**: prefer internal packages to stay focused (config parsing in `internal/config`, orchestration logic in `internal/orchestrator`, etc.). Avoid duplicating logic already expressed in helper packages.
- **Errors & Logging**: use the lightweight logger in `internal/logx`; wrap errors with context using `fmt.Errorf("...: %w", err)`.
- **Documentation**: add Go doc comments for exported types/functions and extend Markdown references (`AGENTS.md`, `SKILL.md`, `docs/*.md`) whenever a feature or flag changes.
- **Dependencies**: this module intentionally avoids external libraries. Discuss any proposed dependency additions in an issue before implementation.

## Testing
- **Unit & integration tests**: `go test ./...` from `dev_agent/` must pass before submitting a PR. Add table-driven tests for new logic, especially within `internal/orchestrator`, `internal/tools`, and configuration parsing.
- **Build check**: run `go build ./...` to verify the CLI compiles across packages.
- **Optional checks**: `go test ./... -race` and `go vet ./...` help catch data races and misuse; run them for complex changes.
- **Coverage expectations**: aim to keep or increase coverage around new paths; justify any gaps in the PR description.

## Pull Request Process
1. Rebase onto the latest `main` before opening the PR.
2. Fill out the PR template (or include in the description): summary, linked issue (e.g., “Fixes #37”), testing commands/results, and any follow-up TODOs.
3. Include updates to docs (`CONTRIBUTING.md`, `AGENTS.md`, etc.) and `worklog.md` when behaviour or processes change.
4. Keep diffs reviewable—split large efforts into multiple PRs when reasonable.
5. Expect at least one maintainer review; address all comments or explain why a suggestion does not apply.
6. CI must be green; if a check is flaky, call it out in the PR before requesting review.

## Communication
- **GitHub Issues**: primary place for new proposals, bug reports, and clarifying questions. Reference the issue ID in commits and PRs.
- **Pull Request comments**: discuss implementation details and review feedback openly; keep the thread updated with any follow-up findings.
- **Discussions / Slack**: if you have access to a shared Pantheon Slack or GitHub Discussions board, post architectural proposals there before investing in large changes. Otherwise use issues for design conversations.

## License
A dedicated `LICENSE` file is not yet published in this repository. By contributing, you agree that your work may be redistributed under the project’s future license once it is formalized. If you require clarification about licensing terms before contributing, please open an issue to discuss it with the maintainers.
