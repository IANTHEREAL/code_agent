# Contributing to dev_agent

Welcome! dev_agent is a Go-based CLI that orchestrates Pantheon MCP agents to
run a disciplined, test-driven workflow. Thoughtful contributions—code, docs, or
issue triage—keep the tool reliable for AI-assisted teams. This guide explains
how to get set up, work effectively, and land changes smoothly.

## Getting Started

### Prerequisites

- Go **1.21+**
- Git plus a GitHub account with access to this repository
- `npx` (bundled with a recent Node.js install) for Markdown linting
- Access credentials for:
  - Azure OpenAI (`AZURE_OPENAI_*` variables)
  - Pantheon MCP endpoint (`MCP_BASE_URL`)
  - GitHub token (`GITHUB_TOKEN`) with repo write scopes for automated pushes
- Git identity variables (`GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`)

### Repository Layout

```text
dev_agent/
├── AGENTS.md          # High-level agent workflow specification
├── SKILL.md           # Skill definitions and contracts
└── dev_agent/         # Go module root (module name: dev_agent)
    ├── cmd/dev-agent  # CLI entry point
    ├── internal/      # brain, config, orchestrator, tools, streaming, logx
    └── docs/          # Design docs (e.g., streaming JSON)
```

All Go commands run from `dev_agent/dev_agent`.

### Setup Steps

1. **Fork and clone**

   ```bash
   git clone git@github.com:<you>/dev_agent.git
   cd dev_agent
   ```

2. **Install Go module dependencies**

   ```bash
   cd dev_agent
   go mod download
   ```

3. **Configure environment**

   - Create a `.env` file at the repo root or export variables in your shell.
   - Required keys (adjust values for your deployment):

     ```env
     AZURE_OPENAI_API_KEY=...
     AZURE_OPENAI_BASE_URL=https://<resource>.openai.azure.com
     AZURE_OPENAI_DEPLOYMENT=<deployment-name>
     MCP_BASE_URL=http://localhost:8000/mcp/sse
     PROJECT_NAME=<workspace friendly name>
     WORKSPACE_DIR=/home/pan/workspace
     GITHUB_TOKEN=<repo write token>
     GIT_AUTHOR_NAME="Your Name"
     GIT_AUTHOR_EMAIL="you@example.com"
     ```

4. **Confirm the CLI builds**

   ```bash
   go build ./cmd/dev-agent
   ```

5. **Optional smoke test**

   ```bash
   go run ./cmd/dev-agent \
     --task "echo hello world" \
     --parent-branch-id <uuid> \
     --project-name "$PROJECT_NAME" \
     --headless
   ```

   Use a safe parent branch ID from Pantheon; the CLI exits early if required
   config is missing.

## Development Workflow

- **Plan first**: Read the GitHub issue, review `AGENTS.md`/`SKILL.md`, and log
  your analysis plus task outline in `/home/pan/workspace/worklog.md`. Update the
  worklog whenever decisions or validations change.
- **Branch naming**: Create feature branches off `main` using a descriptive
  slug, e.g., `pantheon/<type>-<short-topic>` (`pantheon/docs-streaming-guide`).
  Stay consistent so Pantheon history is easy to follow.
- **TDD cadence**: For code, add or update tests before implementation whenever
  possible. For docs, treat linting and completeness checks as your tests.
- **Small, reviewable commits**: Favor incremental commits that tell a story
  (analysis → implementation → validation). Avoid bundling refactors with new
  features.
- **Workspace conventions**: Keep shared artifacts (`worklog.md`,
  `code_review.log`) in `/home/pan/workspace` so orchestration tooling can read
  them. Mention any deviations in your PR.
- **Tooling**: Use standard Go commands—there is no bespoke helper script—so
  reviewers can reproduce results quickly.

## Code Style

- Run `gofmt` or `goimports` on every Go file you touch. Most editors can
  format on save; otherwise:

  ```bash
  gofmt -w path/to/file.go
  ```

- Keep packages cohesive:
  - CLI-specific logic belongs in `cmd/dev-agent`.
  - Reusable orchestration logic lives under `internal/` (e.g.,
    `internal/orchestrator`).
  - New packages under `internal/` should have focused APIs and unit tests.
- Prefer structured logging via `internal/logx`; avoid `fmt.Println` except for
  intentional CLI output.
- Handle errors explicitly; wrap with context using
  `fmt.Errorf("...: %w", err)` so upstream callers see actionable messages.
- Exported functions and types need a short doc comment. Keep inline comments
  sparse and purposeful.
- Update or add docs (`AGENTS.md`, `docs/*.md`, this guide) whenever behavior or
  protocol expectations change.

## Testing

- Run the full unit suite before pushing:

  ```bash
  go test ./...
  ```

- For logic-heavy packages, iterate faster with targeted commands:

  ```bash
  go test ./internal/orchestrator -run TestBranchLifecycle
  ```

- Static analysis:

  ```bash
  go vet ./...
  ```

- When touching CLI behaviors, run a local smoke test (see “Getting Started”)
  and capture the command plus output summary in your PR or worklog.
- If you introduce new environment variables or configuration branches, explain
  how to exercise them and add regression tests when feasible.

## Pull Request Process

1. Ensure your branch is rebased on the latest `origin/main`.
2. Re-run validations (`go test ./...`, `go vet ./...`, `markdownlint` for docs)
   and update `/home/pan/workspace/worklog.md` with:
   - Analysis summary
   - Implemented changes
   - Validation commands and results
   - Branch name plus commit SHA
3. Confirm all required docs are updated (README, design docs, this guide, etc.).
4. Push your branch and open a PR that includes:
   - Linked GitHub issue
   - What changed and why
   - Validation evidence (commands plus pass/fail)
   - Any manual verification details (CLI smoke tests, screenshots)
5. Request review from a maintainer. Be responsive to feedback—update the PR,
   rerun validations, and keep the worklog in sync.
6. After approval, squash or fast-forward merge per maintainer guidance.
   Automated publishing relies on accurate branch metadata, so avoid rewriting
   history once reviews start.

## Issue Guidelines

- **Before filing**: Search existing issues and discussions to avoid duplicates.
- **Bugs**:
  - Provide the CLI command, environment variables (mask secrets), Pantheon
    branch IDs, and relevant excerpts from `worklog.md` or `code_review.log`.
  - List expected versus actual behavior and minimal reproduction steps.
  - Include logs or NDJSON snippets when `--stream-json` is involved.
- **Feature requests**:
  - Describe the workflow pain point, why current tooling is insufficient, and
    any constraints (latency, compliance, security, etc.).
  - Suggest where in the codebase the change might live
    (`internal/orchestrator`, `internal/tools`, docs, etc.).
- **Maintenance chores** (docs, refactors): Tag clearly (e.g., `docs`,
  `tech-debt`) and link related specs when available.

Clear, detailed issues and PRs help maintainers reproduce problems quickly and
keep dev_agent stable. Thanks for contributing!
