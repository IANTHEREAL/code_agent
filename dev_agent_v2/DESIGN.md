# dev-agent v2（Issue Resolve Loop）设计文档

## 背景与动机

现有 `dev_agent`（v1）是一个 **TDD Implement → Review → Fix** 的编排器：外层由 `brain`（LLM）控制流程，内层通过 MCP（Pantheon）创建分支、等待执行完成并收集输出。它还包含一个“最终 publish（commit+push）”阶段。

近期验证的 `pantheon-issue-resolve-loop` workflow 与 v1 的目标不同：

- 强调 **证据优先的有效性验证**（先判断 issue/PR claim 是否成立）
- 以 **单 PR 持续迭代** 为中心（第一次 Fix 创建 PR；后续 Fix 只向同一 PR head 分支 push）
- Review/Verify 的输入输出更偏 **语义**，而不是固定 schema（需要 LLM 控制器保持灵活性）
- 分支执行是长任务（1–2 小时正常），工具层必须稳定等待与错误处理
- **不需要最终 publish**（workflow 自包含 PR 创建、push、检查、merge）

因此 v2 的目标是：保持“LLM 控制器”的灵活性，但在 Go 侧补齐护栏与默认行为，使其能稳定复刻该 workflow。

---

## 设计原则

1. **LLM 控制器优先**：流程控制仍由 `brain` 决策（下一步跑 Validity/Fix/Review/Verify/CI/Merge），避免硬编码有限状态机导致的僵化与 schema 绑定。
2. **工具层强约束与护栏**：用代码兜底“单次调用”“长等待”“失败即停”“分支血缘可追踪”等机械性规则，减少 LLM 偶发跑偏造成的混乱（重复 PR、错误 parent_branch_id 等）。
3. **最小侵入**：尽量复用 v1 的 `internal/brain`、`internal/tools`、`internal/streaming`；新增 v2 的 orchestrator/system prompt 与配置开关即可。
4. **不做 publish**：v2 不再调用任何“最终 push”阶段；所有 git/gh 操作都由 workflow 的 codex exploration 内完成。
5. **最终输出精简**：最终只输出结论与必要指针（issue、PR、Pantheon lineage）；不输出每一步 steps（这些信息在 GitHub/Pantheon/NDJSON 里可追溯）。

---

## 范围（Goals / Non-goals）

### Goals
- 复刻 `pantheon-issue-resolve-loop` 的核心能力：
  - Step 1：同步 master/PR 并验证 issue 有效性（VALID/INVALID）
  - Step 2：Fix/Review/Verify 循环，保持单 PR
  - 可选：CI checks watch、merge（取决于仓库策略/权限）
- 强健的长任务等待：Pantheon branch 1–2h 不算异常。
- headless + 可选 `--stream-json`（NDJSON）事件流。

### Non-goals
- 不实现 `local-tipg-up`（已不存在，忽略）。
- 不强制 codex 输出固定 schema；只做“建议的标记块”，并允许控制器基于语义继续推进。
- 不强制在最终报告中枚举每一步详情（仅输出最终总结与指针）。

---

## 用户接口（CLI）

建议提供一个独立入口，避免影响 v1 行为：

- 二进制：`dev-agent-v2`（推荐），或 `dev-agent --workflow issue-resolve`（兼容模式）

必需参数：
- `--project-name <name>`
- `--parent-branch-id <uuid>`
- `--task <text>`（任务描述，建议包含 issue link 或 existing PR link，以及执行叮嘱）

可选参数：
- `--headless`：与 v1 一致
- `--stream-json`：与 v1 一致（强制 headless）
- `--max-turns`：防止无限循环（默认建议 60）
- 等待策略通过环境变量覆盖（默认对齐 1–2h）：`MCP_POLL_TIMEOUT_SECONDS` / `MCP_POLL_INITIAL_SECONDS` / `MCP_POLL_MAX_SECONDS` / `MCP_POLL_BACKOFF_FACTOR`

最终输出（stderr 打印 pretty JSON，保持 v1 风格）：
- `status`: `completed` / `FINISHED_WITH_ERROR` / `iteration_limit`
- `summary`: 最终结论（VALID/INVALID、PR 是否可合并、merge 是否完成、阻塞原因等）
- `task`
- `pr_url` / `pr_number` / `pr_head_branch`（如果存在）
- `start_branch_id` / `latest_branch_id`
- `instructions`：由 brain 在收尾阶段生成的人类下一步指引（例如“去 Pantheon 看 manifest <id> / 继续从 latest_branch_id 跑 / 看 PR checks”）

---

## 体系结构

### 总览

v2 沿用 v1 的两层结构，但替换 system prompt 与终止逻辑：

- **LLM 控制器（Orchestrator）**：`brain.Complete(messages, tools)` 生成“下一步要调用的工具”或“最终报告 JSON”
- **执行器（ToolHandler）**：执行工具调用（核心是 `execute_agent`），阻塞等待 Pantheon 分支完成并返回输出

关键差异：
- v2 **不执行 publish**（删除/关闭 `finalizeBranchPush` 触发路径）
- v2 需要 **新的 system prompt**（等价于 skill 文本 + 运行约束）
- v2 需要 **新的 iteration 计数策略**（不能只依赖 `review_code` 次数）

### 推荐代码布局（提案）

```
dev_agent_v2/
  DESIGN.md

dev_agent/
  cmd/
    dev-agent-v2/
      main.go              # 新入口（或在 v1 main.go 增加 --workflow）
  internal/
    orchestrator_v2/
      orchestrator.go      # v2 的 Orchestrate（复用 brain/tools/streaming）
      prompt.go            # v2 system prompt + 初始 user payload
      parse.go             # 解析最终 JSON（与 v1 类似）
```

复用（不动或最小改动）：
- `internal/brain`：Azure OpenAI chat completions
- `internal/tools`：`execute_agent`/`branch_output`/`read_artifact`（但需要调整默认 poll timeout 到长任务）
- `internal/streaming`：NDJSON 事件（可选）

---

## 工具面与执行语义

### 继续保留的工具

v2 仍只暴露 3 个工具给 LLM 控制器：

- `execute_agent(agent, prompt, project_name, parent_branch_id)`
- `branch_output(branch_id, full_output?, tail?, max_chars?)`
- `read_artifact(branch_id, path)`

其中 `execute_agent` 的语义保持 “spawn + wait + return output”，因此 playbook 不需要 `get_branch`。

### 等待策略（必须升级为长任务默认）

`ToolHandler.checkStatus` 的默认等待参数需要对齐 1–2h：

- `poll_timeout` 默认值建议：`3h`（>= 2h）
- `poll_max` 上限建议：`600s`
- backoff 仍可保留（指数退避到 600s）

> 备注：这是 v2 稳定性的底座；否则会出现“合法长任务被误判为超时失败”的糟糕输出。

### 关于 branch_output 全量回填的问题

当前 v1 的 `execute_agent` 会在返回结果中携带 `branch_output(full=true)` 的文本（用于给控制器做语义决策）。这在极长输出时可能带来上下文膨胀风险。

按本次需求：v2 **直接实现了截断/提示机制**，避免长跑任务把全量日志塞回 LLM messages：

- `execute_agent` 返回：
  - `response`：输出的 tail excerpt（默认 8k chars）
  - `response_truncated=true/false`
  - `full_output_hint`：提示用 `branch_output` 按需拉更多（配合 `tail/max_chars`）
- `branch_output` 支持：
  - `tail=true` + `max_chars=<n>`：返回尾部 excerpt（handler 可能内部拉 full_output 再截断）
  - 返回 `output` + `output_truncated=true/false`

因此：长输出仍可在需要时被“二次拉取”，但不会默认污染对话上下文。

---

## v2 Playbook（System Prompt）设计

v2 的 system prompt 需要把 `pantheon-issue-resolve-loop` 的流程与关键约束“写死”，但不要求每一步输出固定 schema。

必含的控制器规则（建议强制）：

1. **Single Call Per Turn**：每次 assistant 回复必须满足以下二选一：
   - 发起 **恰好 1 个** tool call；或
   - 输出最终 JSON 报告（`is_finished=true`）并停止
2. **One Fix Run at a Time**：不得并发发起探索；等待 `execute_agent` 返回后才能进入下一步。
3. **Single PR**：第一次 Fix 创建 PR；后续 Fix 必须 checkout 现有 PR head 并 push，不得创建新 PR。
4. **Evidence-first**：没有 VALID 结论前不得进入 Fix。
5. **Long-running expectation**：Pantheon 任务可能需要 1–2h，属于正常；不要把等待当作卡死。

初始 user payload（JSON）建议显式包含：
- `task`
- `project_name`
- `workspace_dir`
- `parent_branch_id`
- `notes`：重复关键规则（single-call、single-PR、no-publish）

> 说明：v2 仍然依赖 LLM 在语义上维护变量（`pr_number/pr_url/pr_head_branch/last_fix_branch_id` 等）。Go 侧只负责把 branch lineage（start/latest）附加到最终 report，并不强制解析 codex 的语义输出。

另外，为避免 system prompt 与文档 drift，v2 推荐在运行时直接读取 `dev_agent_v2/SKILL.md` 作为 playbook 注入到 system prompt（找不到时再回退到内置简版说明）。

---

## 终止条件与结果报告

v2 的最终报告格式建议与 v1 保持兼容（ParseFinalReport 仍然只认 JSON）：

```json
{
  "is_finished": true,
  "status": "completed|FINISHED_WITH_ERROR|iteration_limit",
  "summary": "...",
  "task": "...",
  "pr_url": "...",
  "pr_number": 123,
  "pr_head_branch": "..."
}
```

差异点：
- **不输出 steps**（不列每一轮/每一步），只输出最终结论与关键指针。
- Go 侧仍会附加 `start_branch_id` / `latest_branch_id`；`instructions` 由 brain 生成。

---

## 可靠性护栏（必须由代码实现，不能只靠 prompt）

1. **单 tool-call 强制**：如果 LLM 返回 `len(tool_calls) != 1` 且不是最终报告，则拒绝执行并回填错误，让 LLM 重试。
2. **全局 iteration / execution 限制**：按 turn 数或 `execute_agent` 次数计数，达到上限输出 `iteration_limit` 总结并停止（防止无限循环）。
3. **分支失败即停**：`checkStatus` 发现 branch `failed`，直接返回 `FINISHED_WITH_ERROR`，并给出 Pantheon manifest 指引。
4. **无 publish**：v2 路径中不得调用任何 publish/finalize 逻辑。

---

## 风险与后续改进（出现问题再做）

1. **上下文膨胀（branch_output 太长）**
   - 现状：v2 已通过 `response_truncated/full_output_hint + branch_output(tail/max_chars)` 解决“默认全量回填”的问题。
   - 后续改进（如果需要）：在 MCP server 侧支持真正的 server-side `tail/max_chars`，避免 handler 为了截断而拉 full_output。

2. **LLM 语义抽取 PR 信息不稳定**
   - 症状：LLM 没能可靠记住/输出 `pr_number/pr_url`。
   - 后续改进：在控制器 prompt 中要求“每次涉及 PR 的阶段末尾输出一个短的 `PR:` 信息块”；或在 Go 侧做轻量正则抽取作为补充（不作为强依赖）。

3. **PR 重复创建**
   - 症状：Fix 阶段错误地又创建了一个 PR。
   - 后续改进：在 prompt 中更强地约束“如果已有 PR，则必须 checkout 并 push”；并在 Go 侧记录 `pr_number` 后，后续 Fix prompt 自动注入“禁止新建 PR”与“必须 checkout <pr_number>”。

---

## 实施计划（最小可行切片）

1. 新增 `cmd/dev-agent-v2`（或 `--workflow issue-resolve`），使用 v2 system prompt 启动 orchestrator。
2. 在 v2 orchestrator 中移除 publish 步骤；更新 iteration 计数与单 tool-call 强制。
3. 更新 `internal/config`：移除 `GITHUB_TOKEN/GIT_AUTHOR_*` 的读取/校验（v2 本身不做 publish；gh/git 认证由 Pantheon 运行环境在 exploration 内处理）。
4. 更新 `internal/tools`：将默认 poll timeout/max poll 对齐长任务（>= 2h，max=600s）。
5. 增补单测：多 tool-call 拒绝执行、iteration limit、超时默认值、无 publish 路径。
