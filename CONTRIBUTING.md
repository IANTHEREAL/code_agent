# Contributing to dev_agent

We are excited that you want to help improve **dev_agent**, a Go CLI that automates a disciplined *Implement → Review → Fix* (TDD) workflow via specialist MCP agents. This guide explains how to get set up, develop features, and submit high-quality contributions.

---

## 1. Prerequisites

### Tooling
- **Go 1.21** (or newer patch release). The module is tested with Go 1.21; other versions are unsupported.
- **Git** with access to your GitHub account (SSH or HTTPS).
- Ability to run CLI binaries on Linux/macOS (the repo's workspace defaults to `/home/pan/workspace`).

### Required environment variables
The CLI loads a `.env` file at the repo root (if present) and then reads your shell environment. These variables are required by `internal/config`:

| Variable | Purpose |
| --- | --- |
| `AZURE_OPENAI_API_KEY` | Azure OpenAI API key. |
| `AZURE_OPENAI_BASE_URL` | HTTPS endpoint for your Azure OpenAI resource (e.g., `https://my-resource.openai.azure.com`). |
| `AZURE_OPENAI_DEPLOYMENT` | Deployment/model name to call. |
| `AZURE_OPENAI_API_VERSION` | Optional; defaults to `2024-12-01-preview`. |
| `MCP_BASE_URL` | MCP server URL; defaults to `http://localhost:8000/mcp/sse`. |
| `MCP_POLL_INITIAL_SECONDS`, `MCP_POLL_MAX_SECONDS`, `MCP_POLL_TIMEOUT_SECONDS`, `MCP_POLL_BACKOFF_FACTOR` | Optional polling tunables (defaults enforce sane exponential backoff). |
| `PROJECT_NAME` | Human-friendly project identifier surfaced in prompts/reports (required unless `--project-name` flag is provided). |
| `WORKSPACE_DIR` | Workspace root (defaults to `/home/pan/workspace`). |
| `GITHUB_TOKEN` | Personal access token with repo permissions so you can push your feature branches to your fork. |
| `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL` | Your identity for commit authorship. |

> Tip: Keep secrets out of commits. Use `.env` locally and rely on deployment secrets for CI/CD.

---

## 2. Development Setup
1. **Fork** the repository on GitHub and clone your fork:
   ```bash
   git clone git@github.com:<you>/dev_agent.git
   cd dev_agent
   ```
2. **Check out** the Go module directory (source code lives in `dev_agent/`):
   ```bash
   cd dev_agent
   go mod download
   ```
3. **Create a `.env`** (optional) to mirror your environment:
   ```bash
   cat > .env <<'EOF'
   AZURE_OPENAI_API_KEY=...
   AZURE_OPENAI_BASE_URL=https://<resource>.openai.azure.com
   AZURE_OPENAI_DEPLOYMENT=<deployment-name>
   AZURE_OPENAI_API_VERSION=2024-12-01-preview
   MCP_BASE_URL=http://localhost:8000/mcp/sse
   PROJECT_NAME=<your-project>
   WORKSPACE_DIR=/home/pan/workspace
   GITHUB_TOKEN=ghp_...
   GIT_AUTHOR_NAME="Your Name"
   GIT_AUTHOR_EMAIL=you@example.com
   EOF
   ```
4. **Run the CLI** (optional smoke test):
   ```bash
   go run ./cmd/dev-agent --help
   ```
5. Keep `/home/pan/workspace/worklog.md` and `/home/pan/workspace/code_review.log` up to date during multi-phase development—they are consumed by the automated agents.

---

## 3. Project Structure

```
dev_agent/
├── AGENTS.md              # High-level agent orchestration spec
├── SKILL.md               # Skill usage / CLI instructions
├── dev_agent/             # Go module root
│   ├── cmd/dev-agent/     # CLI entrypoint
│   ├── internal/          # Core packages
│   │   ├── brain          # Azure OpenAI client wrapper
│   │   ├── config         # Env + workspace configuration
│   │   ├── logx           # Lightweight logging helpers
│   │   ├── orchestrator   # Implement → Review → Fix loop
│   │   └── tools          # MCP integration + branch tooling
│   └── docs/              # Design docs (e.g., streaming JSON)
```

When referencing paths in discussions or PRs, prefer module-relative paths (e.g., `dev_agent/internal/orchestrator`).

---

## 4. Building & Testing
- **Unit tests**: run from the Go module root.
  ```bash
  cd dev_agent
  go test ./...
  ```
- **Formatting**: `gofmt -w <files>` before committing. Go tooling enforces tabs for indentation and spaces for alignment.
- **Static checks** (optional but encouraged): `go vet ./...`, `staticcheck ./...`.
- **CLI validation**: for features that impact runtime behavior, run `go run ./cmd/dev-agent --task "..."`
  with representative `--parent-branch-id` and env vars.

Always run `go test ./...` before pushing or opening a PR; CI expects a clean test run.

---

## 5. Contribution Workflow
1. **Discuss the change**: file an issue (or comment on an existing one) describing the bug/feature, expected behavior, and any context (logs, configs, branch IDs).
2. **Create a branch** from the latest `main` (examples: `pantheon/feat-<topic>`, `pantheon/fix-<bug>`). Use descriptive names; avoid committing directly to `main`.
3. **Follow the Implement → Review → Fix loop**:
   - *Implement*: write the minimal code/docs/tests to satisfy the requirement. Update `worklog.md` with what you touched.
   - *Review*: self-review or leverage MCP review agents. Capture review findings in `code_review.log` to keep the automation in sync.
   - *Fix*: address review items, update tests, and reiterate until no blocking issues remain.
4. **Commit** logical chunks with clear messages (imperative mood, <72 character subject). Sign commits if your org requires it.
5. **Open a PR** against `main` on GitHub:
   - Reference the issue number (`Fixes #37`).
   - Summarize the change, validation steps (`go test ./...`, manual CLI checks), and any new configuration or docs.
   - Link to relevant artifacts (screenshots, logs, Pantheon branch IDs).
6. **Respond to review** promptly. Update the PR by amending or adding commits; keep the history clean if requested.

---

## 6. Implement → Review → Fix (TDD Automation Model)
- The orchestrator enforces a strict loop via `claude_code` (implementation) and `review_code` (automated review). Humans should mimic this discipline:
  - Treat each iteration as a mini sprint—implement tests + code, capture results, then pause for review.
  - Use `worklog.md` to record each phase (Phase 0 context check, Phase 1 design, Phase 2 implementation/validation).
  - Record review feedback (automated or human) in `code_review.log` so the agent can reason about outstanding issues.
- Publishing a branch requires **zero P0/P1 findings**. Do not request a publish/push until reviews are green and tests are passing.
- Keep test automation fast; prefer table-driven Go tests and limit external dependencies to mocks/fakes.

---

## 7. Code Style & Conventions
- Follow idiomatic Go style (`gofmt`, `goimports`). Keep functions small and focused.
- Exported symbols need GoDoc comments when they are part of the public/CLI API surface.
- Avoid introducing new third-party dependencies unless necessary. If required, justify them in the PR description.
- Prefer standard library logging patterns via `internal/logx`.
- Update or create documentation (AGENTS, SKILL, docs/stream-json.md, CONTRIBUTING) whenever behavior changes.
- Keep Markdown ASCII-only unless quoting user-visible strings. Wrap at ~100 characters for readability.

---

## 8. Issues & Pull Requests
- **Bug reports** should include: reproduction steps, expected vs. actual behavior, CLI flags used, logs (especially final JSON output), and any MCP/branch IDs.
- **Feature requests** should outline the problem, proposed solution, API/UX impact, and testing strategy.
- **Pull requests** must list:
  - What changed and why.
  - Tests run (`go test ./...`, manual steps).
  - Follow-up work (if any).
  - Screenshots or JSON snippets when user-facing output changes.

Use GitHub Discussions or Issues for questions; avoid DM-only conversations so context stays public.

---

## 9. Contact & Support
- **Issues**: https://github.com/IANTHEREAL/dev_agent/issues
- **Maintainer**: @IANTHEREAL (repository owner). Mention in issues/PRs for urgent questions.
- **Security concerns**: Do **not** open a public issue. Email the maintainer privately or use GitHub Security Advisories.

Thank you for helping us automate high-quality TDD workflows!
