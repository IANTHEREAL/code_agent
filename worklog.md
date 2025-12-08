## Phase 1 – Analysis & Design

### Context / RCA
- `internal/tools/handler.go` hardcodes `timeout = 1800.0` inside `checkStatus`, so every branch poll aborts after 30 minutes regardless of task complexity.
- The rest of the system already externalizes polling configuration through env-driven `AgentConfig`, but the status timeout bypasses that config which makes it impossible to tune for long-running jobs.
- Because `checkStatus` owns the polling loop, the timeout must be plumbed through `ToolHandler` so callers (or configuration) can influence the limit without rewriting orchestration logic.

### Plan
1. **Configuration surface**: introduce `BRANCH_STATUS_TIMEOUT_SECONDS` (default 1800) parsed in `internal/config/FromEnv` and stored on `AgentConfig`. This keeps compatibility (default matches old behavior) while allowing env overrides; we can still add CLI flag later without changing handler logic because the handler will accept a duration parameter.
2. **ToolHandler wiring**: extend `ToolHandler`/`NewToolHandler` with a `branchStatusTimeout time.Duration`. When zero/negative we fall back to 30 minutes for backward compatibility. `checkStatus` will use this value unless the caller explicitly passes `timeout_seconds`, so the legacy argument-based override still works.
3. **Testability improvements**: add an injectable clock (defaulting to `time.Now`/`time.Sleep`) so tests can simulate long waits instantly. New tests will assert (a) default timeout matches 30 minutes when config/env unset, (b) env override takes effect, and (c) `checkStatus` waits long enough for slow branches instead of failing immediately.
4. **TDD flow**: add failing tests covering the scenarios above (config parsing + handler timeout behavior). Then implement the configurable timeout, rerun the tests, and document the results here in the worklog.

## Phase 2 – Implementation & Verification
- Added config + handler tests first (`internal/config/config_test.go`, new handler tests with fake clock) to cover default timeout, env override, and slow-branch success; they failed to compile before the implementation which satisfied the TDD requirement.
- Implemented `BRANCH_STATUS_TIMEOUT_SECONDS` in `AgentConfig`, plumbed it through `NewToolHandler`, and added `branchStatusTimeout` plus an injectable clock so `checkStatus` can rely on configurable defaults while staying backward compatible with the `timeout_seconds` argument.
- Verified behavior with `go test ./...` (covers config + tools packages); all suites pass locally in ~0.5s.
