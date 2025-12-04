# Contributing to dev_agent

## 1. Welcome & Overview
- Thanks for your interest in dev_agent — an AI agent development tool written in Go that orchestrates autonomous coding workflows.
- Contributions of any size are valued, from typo fixes to new skills or orchestration capabilities.
- This guide explains the end-to-end process so every change is high-quality, tested, and easy to review.

## 2. Getting Started
### Prerequisites
- Go 1.21 or newer (`go version` to confirm).
- Git and a GitHub account with permission to fork or push branches.
- Access credentials for the runtime integration: `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_BASE_URL`, `AZURE_OPENAI_DEPLOYMENT`, `AZURE_OPENAI_API_VERSION`, `GITHUB_TOKEN`, `GIT_AUTHOR_NAME`, and `GIT_AUTHOR_EMAIL`. Optional but recommended: `PROJECT_NAME`, `WORKSPACE_DIR`, and MCP tuning env vars (see `internal/config`).

### Clone & Setup
```bash
git clone https://github.com/IANTHEREAL/dev_agent.git
cd dev_agent
go mod download
```

### Local Execution
- Run the CLI main entry point with your task and branch identifiers:
  ```bash
  go run ./cmd/dev-agent \
    --task "short description" \
    --parent-branch-id "<uuid>" \
    --project-name "my-project"
  ```
- Review `docs/` for feature-specific notes such as streaming JSON output.

## 3. Development Workflow
- **Branch naming:** Use descriptive, kebab-case prefixes, e.g., `feature/37-streaming-ui`, `bugfix/publish-panic`, or `docs/contributing`. For automation-driven changes, follow the orchestration’s generated branch names.
- **Commit conventions:** Keep commits atomic, present-tense, and include the GitHub issue number (e.g., `Add streaming JSON docs (#42)`). Multi-line messages should summarize what and why.
- **Sync frequently:** Rebase or merge `main` before opening a PR to reduce conflicts. Avoid force-pushing shared branches.
- **Worklog etiquette:** If you use automated worklogs, keep them up to date so reviewers can follow reasoning.

## 4. Code Style
- Follow idiomatic Go style as enforced by `gofmt` and `goimports`. Run them before committing.
- Prefer standard library packages when possible; keep third-party dependencies minimal and add them via `go mod tidy`.
- Validate logic with `go vet ./...` and, when available, `golangci-lint run`. Install linters via `go install` if needed.
- Document exported types/functions, especially in `internal/brain`, `internal/tools`, and other packages that orchestrate agents.

## 5. Testing
- Run the full unit test suite before submitting:
  ```bash
  go test ./...
  ```
- Add focused tests alongside new code (e.g., `internal/orchestrator`, `internal/tools`, `cmd/dev-agent`). Tests should cover happy paths and failure handling (timeouts, bad config, etc.).
- When touching integration points (Azure, MCP, GitHub), prefer interface-driven designs with fakes so tests remain deterministic.
- Consider adding examples or doc tests for complex configuration behaviors.

## 6. Pull Request Process
- **Before opening a PR:**
  - Ensure commits are rebased on the latest `main`.
  - Confirm `go fmt`, `go vet`, linters, and `go test ./...` pass.
  - Update documentation (README, docs/, config comments) when behavior changes.
- **PR description checklist:**
  - Link the related issue (`Resolves #37`).
  - Summarize the change, risks, and any follow-up tasks.
  - Attach screenshots or terminal captures if UI/CLI output changed.
- Expect at least one maintainer review. Address feedback with follow-up commits (avoid force-push unless asked).

## 7. Issue Guidelines
- **Bug reports:** Include reproduction steps, observed vs. expected behavior, CLI args/env vars, and relevant logs (redact secrets). Mention commit SHA if not on `main`.
- **Feature requests:** Explain the workflow you want to unlock for dev_agent users (agents, MCP skills, orchestration improvements). Provide acceptance criteria or pseudo-code if possible.
- **Good first issues:** Label simple, well-scoped tasks so newcomers can onboard quickly.
- Check existing issues before filing new ones, and comment if you plan to take an open task.

## 8. Code of Conduct
- dev_agent follows the [Contributor Covenant v2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). Be respectful, assume good intent, and escalate concerns to the maintainers via GitHub discussions or issues.
- Harassment, discrimination, or disruptive behavior is not tolerated. Report incidents to the project maintainers immediately.

Thanks again for contributing—your improvements help dev_agent build safer, more capable AI development agents!
