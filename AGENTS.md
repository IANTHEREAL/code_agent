# Dev Agent Specification

## Overview
- Purpose: automate a disciplined Test-Driven Development workflow by delegating implementation and review tasks to specialist MCP agents (`claude_code` and `codex`).
- Entry point: `cmd/dev-agent/main.go` CLI binary. Outputs a structured JSON report summarizing task outcome and observed branch lineage.
- Operating mode: headless (`--headless` flag) or interactive chat loop mirroring all assistant/tool exchanges to stdout.

## Inputs and Configuration
- Required CLI args: `--parent-branch-id` (source branch UUID); optional `--task`, `--project-name`.
- Prompted input: if `--task` omitted, CLI requests a task description over stdin.
- Environment variables (loaded via `internal/config`):
  - `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_BASE_URL`, `AZURE_OPENAI_DEPLOYMENT`, optional `AZURE_OPENAI_API_VERSION`.
  - `MCP_BASE_URL` (defaults to `http://localhost:8000/mcp/sse`).
  - Polling control: `MCP_POLL_INITIAL_SECONDS`, `MCP_POLL_MAX_SECONDS`, `MCP_POLL_TIMEOUT_SECONDS`, `MCP_POLL_BACKOFF_FACTOR`.
  - Workspace context: `PROJECT_NAME`, optional `WORKSPACE_DIR` (defaults `/home/pan/workspace`).
  - `GITHUB_TOKEN` for final branch publish.
- Optional `.env` file at repo root is loaded first, non-destructively.

## High-Level Workflow
1. CLI gathers configuration, normalises project/task values, and constructs initial conversation messages (`BuildInitialMessages`).
2. Orchestrator loop uses `LLMBrain` (Azure OpenAI Chat Completions) to drive a system prompt enforcing Implement → Review → Fix.
3. Every `execute_agent` call triggers `MCPClient.parallel_explore`, cloning from the tracked parent branch and creating a new branch lineage.
4. `ToolHandler` polls branch status until completion, recording results and updating stored branch IDs.
5. Once `codex` reports zero P0/P1 issues (or iteration limit reached), orchestrator issues a publish prompt to `claude_code` to commit and push the workspace.
6. Final JSON report includes completion status, summary, original task, branch lineage, and publish metadata (`publish_report`, `publish_pantheon_branch_id`).

## Component Responsibilities
- `internal/config.AgentConfig`:
  - Validates required env vars, normalises URLs, and enforces consistent polling backoff constraints.
  - Supplies workspace filename defaults (e.g., `worklog.md`).
- `internal/brain.LLMBrain`:
  - Wraps Azure Chat Completion HTTP calls with exponential backoff, logging, and tool definitions support.
  - Caps responses via `MaxCompletionTokens` (4000) and reuses deployment name for model identifier.
- `internal/orchestrator`:
  - Defines the system prompt containing agent roles, workflow constraints, and publish instructions.
  - Maintains conversation history, tracks review iterations (max 8), and identifies final JSON reports.
  - `finalizeBranchPush` sends a structured prompt to `claude_code` to run git commit/push using supplied GitHub token.
  - Offers both non-interactive (`Orchestrate`) and interactive (`ChatLoop`) flows.
- `internal/tools.MCPClient`:
  - Implements JSON-RPC over HTTP/SSE with retries, session identifiers, and unified result normalisation.
  - Exposes helper methods: `ParallelExplore`, `GetBranch`, `BranchReadFile`.
- `internal/tools.ToolHandler`:
  - Dispatches tool calls from the LLM to MCP actions.
  - Maintains branch lineage via `BranchTracker`, exposes `BranchRange` for final reporting.
  - Provides `read_artifact` for fetching review logs before Fix phases.
- `internal/logx`:
  - Lightweight log utility controlling log level and formatting for stdout/stderr.

## External Systems and Contracts
- **Azure OpenAI**: expects Azure-specific REST endpoint; API key provided via header `api-key`.
- **Pantheon MCP**: accessible via SSE-enabled JSON-RPC; agents named `claude_code` and `codex` must exist remotely.
- **GitHub**: branch publish prompt assumes git commands can run inside agent execution, authenticated using `GITHUB_TOKEN`.

## Observability and Reporting
- Logging: informational progress (`LLM iteration`, MCP requests) routed through `logx`.
- Work artifacts expected at `/home/pan/workspace/worklog.md` and `/home/pan/workspace/codex_review.log` to coordinate Implement/Fix/Review phases.
- Final CLI output: pretty-printed JSON that always includes `task`, `summary`, `status`, `is_finished`, `start_branch_id`, `latest_branch_id`, and an `instructions` string. The instructions summarize how to act on the result (e.g., inspect the latest Pantheon branch/manifest, read the `publish_report` to find the GitHub branch, or—when `status` is `iteration_limit`—choose between rerunning dev-agent with the latest branch ID or taking manual action). When publishing succeeds the payload also carries `publish_report` and `publish_pantheon_branch_id`. The publish step now enforces a mandatory report from the agent describing the GitHub repository, branch, commit hash, and where to find the implementation/test logs; missing that report fails the publish step.

## Failure Modes and Safeguards
- Configuration errors abort before orchestration starts with descriptive stderr messages.
- MCP and Azure calls retry with exponential backoff (capped attempts) and log warnings on intermediate failures.
- Branch polling enforces timeout via configurable intervals; on timeout or iteration exhaustion, publishing still attempts to push current state.
- Unsupported tool calls or malformed arguments return structured `"status": "error"` payloads to the orchestrator.
