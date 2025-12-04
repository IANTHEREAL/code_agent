# Contributing to dev_agent

Thanks for helping make dev_agent better! The project relies on a thoughtful community of contributors who share agent-building ideas, improve the Go codebase, and document evolving workflows. This guide walks you through everything needed to get productive quickly.

## Getting Started

### Prerequisites
- Go 1.21+ (use the latest patch release when possible)
- Git 2.40+ with an SSH or HTTPS GitHub configuration
- A POSIX shell (the repo is developed and tested on Linux/macOS)
- Optional but helpful: a GOPATH-aware editor with Go tooling support (VS Code + `gopls`, GoLand, etc.)

### Local setup
1. Fork [IANTHEREAL/dev_agent](https://github.com/IANTHEREAL/dev_agent) to your GitHub account.
2. Clone your fork and move into the repository root:
   ```bash
   git clone git@github.com:<your-user>/dev_agent.git
   cd dev_agent
   ```
3. The Go module lives in the nested `dev_agent/` directory. Install dependencies and verify the build from there:
   ```bash
   cd dev_agent
   go mod tidy
   go test ./...
   ```
4. Return to the repository root (`cd ..`) for documentation work or when following the workflow below.

## Development Workflow
- Always start from an up-to-date `main`: `git checkout main && git pull origin main`.
- Create descriptive branches using the format `yourname/issue-<id>-<short-topic>`.
- Keep commits focused. Each commit should compile, pass tests, and describe *why* the change exists (e.g., `Add streaming transport guard`).
- Reference the relevant GitHub issue in commit messages and PR descriptions (`Fixes #37`).
- Push early and often so reviewers can follow along, but rebase to keep history tidy before requesting review.

## Code Standards
- Follow idiomatic Go style: small interfaces, prefer composition, avoid global state, and keep exported APIs well commented.
- Format code with `gofmt -w` (or `go fmt ./...`) and organize imports via `goimports` or your editor before committing.
- Run lightweight static checks—`go vet ./...` and `golangci-lint run` if you have it installed.
- Name packages for what they provide (`brain`, `orchestrator`, `streaming`, `tools`, etc.) and files for what they implement.
- Keep configuration, logging, and streaming helpers in their respective `internal/*` packages. Avoid circular dependencies; prefer injectible interfaces.
- Document new behavior alongside the code (comments or updates to `AGENTS.md`, `SKILL.md`, or `docs/stream-json.md`).

## Testing
- Execute the full suite from the module root: `cd dev_agent && go test ./...`.
- Use table-driven tests for orchestration logic and tool integrations; place them next to the code under test (e.g., `internal/brain/..._test.go`).
- When adding a new subsystem, include at least one integration-style test that validates a realistic flow through the orchestrator or streaming layers.
- Tests should be deterministic and avoid hitting external services; rely on fakes in `internal/tools` or create new ones.
- Update docs to describe any new test commands or data files you introduce.

## Pull Request Process
- ✅ Ensure `go fmt`, `go vet`, and `go test ./...` all pass.
- ✅ Update or add documentation and examples when behavior changes.
- ✅ Rebase onto the latest `main` to remove merge conflicts before requesting review.
- ✅ Fill out the PR template with context, screenshots/recordings (if UI/CLI output matters), and explicit test evidence.
- Expect at least one maintainer review. Please respond to feedback within a couple of business days; push follow-up commits that address comments clearly.
- Once approved, maintainers will handle merging (squash-merge by default). Do not force-push after approval unless requested.

## Issue Reporting Guidelines
- Search existing issues (open and closed) to avoid duplicates.
- Provide clear reproduction steps, including CLI flags, environment variables, and sample payloads when the orchestrator or streaming JSON flows are involved.
- Attach relevant logs with timestamps/levels and strip secrets before sharing.
- For feature ideas, explain the motivating use case, which agent types it impacts, and the desired UX for CLI or API consumers.
- Tag issues with the closest area (`brain`, `orchestrator`, `streaming`, `tools`, `docs`) to help triage.

## Project Structure Overview
```
repo root /
├── AGENTS.md, SKILL.md        # High-level docs about agents and skills
├── CONTRIBUTING.md            # (this guide)
└── dev_agent/                 # Go module
    ├── cmd/dev-agent/         # Entry point binary, CLI wiring
    ├── docs/stream-json.md    # Streaming format details
    ├── internal/brain         # Core agent reasoning loop
    ├── internal/config        # Config loading, env parsing
    ├── internal/logx          # Logging helpers
    ├── internal/orchestrator  # Task orchestration + planning
    ├── internal/streaming     # SSE/Web transport adapters
    └── internal/tools         # Built-in tool implementations
```

Questions? Open a discussion or reach out in the issue you are tackling—maintainers are happy to help you get a change across the finish line. Welcome aboard!
