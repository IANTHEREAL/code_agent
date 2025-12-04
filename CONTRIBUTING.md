# Contributing to dev_agent

Thanks for helping improve the Pantheon dev agent. We aim to keep the project welcoming, reliable, and well-documented—please read this guide before opening a pull request.

## Getting Started / Dev Setup

1. Fork the repository and clone your fork.
2. Move into the Go module that lives under the nested `dev_agent` directory:

   ```bash
   cd dev_agent/dev_agent
   go mod download
   ```

3. Ensure Go 1.21+ is installed and available on your `PATH`.
4. Configure the Azure OpenAI + MCP credentials described in [AGENTS.md](AGENTS.md) and the CLI usage notes in [SKILL.md](SKILL.md). The quickest way is to create a `.env` at the repo root so `internal/config` picks it up automatically.
5. Run the CLI once to confirm your environment:

   ```bash
   go run ./cmd/dev-agent --task "hello world" --parent-branch-id <uuid> --headless
   ```

6. Install any optional tooling you plan to use (`golangci-lint`, `goimports`, etc.).

## Code Style & Conventions (Go-specific)

- Always run `gofmt -w` (or `goimports`) on touched files; CI reviewers assume source is formatted.
- Follow the package layering already used in `internal/*`: keep orchestration logic in `internal/orchestrator`, configuration in `internal/config`, and avoid new circular dependencies.
- Pass `context.Context` as the first parameter for long-running or network-bound functions, and propagate deadlines instead of inventing your own timers.
- Prefer explicit errors (`fmt.Errorf("context: %w", err)`) and keep log output going through `internal/logx` so verbosity flags continue to work.
- Keep functions small and unit-testable; when in doubt, look at the patterns in `cmd/dev-agent` and the MCP tooling under `internal/tools`.
- Update [AGENTS.md](AGENTS.md) or [SKILL.md](SKILL.md) when you make behavior changes that affect the documented agent workflow or CLI UX.

## Pull Request Process

1. Create a branch off `main` using the `pantheon/issue-<id>-<slug>` convention (or similar descriptive slug).
2. Keep pull requests scoped; mechanical refactors should land separately from feature work.
3. Include context in the description: the problem, how you solved it, and any follow-up TODOs.
4. Attach screenshots or JSON snippets when modifying CLI output so reviewers can reason about UX.
5. Ensure docs and samples stay in sync (update AGENTS/SKILL/`docs/*` as needed).
6. Run the full test suite and lint tools before pushing.

## Testing Guidelines

- Run unit tests locally from the module root:

  ```bash
  cd dev_agent/dev_agent
  go test ./...
  ```

- Targeted packages (e.g., orchestration) can be run with `go test ./internal/orchestrator -run TestName`.
- For manual end-to-end verification, follow the CLI recipes in [SKILL.md](SKILL.md) using realistic `--task` and `--parent-branch-id` values; capture the JSON report for reviewers if behavior changes.
- When adding new agents, tools, or config flags, add regression tests alongside the code (table-driven tests are preferred) and document expected environment variables in [AGENTS.md](AGENTS.md).

## Issue Reporting

- Use GitHub Issues and include:
  - A concise summary and reproduction steps.
  - The command you ran (`dev-agent` vs `dev-agent-chat`) and whether you followed the [SKILL.md](SKILL.md) workflow.
  - Relevant environment values (omit secrets) and excerpts from the CLI JSON report or `worklog.md`.
  - Expected vs. actual behavior plus any temporary workarounds you discovered.
- Tag issues as `bug`, `enhancement`, or `docs` so we can prioritize appropriately.

## Project Structure Overview

- `AGENTS.md` – architectural spec for the two-agent TDD workflow; read this before changing orchestration logic.
- `SKILL.md` – quick-start skill card showing how to run the CLI and which inputs it expects.
- `dev_agent/cmd/dev-agent` – main CLI entry point that wires config, orchestration, and reporting.
- `dev_agent/internal/config` – environment + `.env` parsing and validation (all new settings belong here).
- `dev_agent/internal/orchestrator` – governs the implement → review → fix loop and publish flow.
- `dev_agent/internal/tools` – Pantheon MCP integrations (`parallel_explore`, branch tracking, file reads).
- `dev_agent/internal/brain`, `internal/logx`, `internal/streaming` – supporting packages for LLM calls, logging, and incremental output.
- `dev_agent/docs/` – reference docs (e.g., `stream-json.md`) that explain auxiliary protocols and should be updated when output formats change.

We’re excited to collaborate—feel free to open a draft PR early if you’d like feedback on direction.
