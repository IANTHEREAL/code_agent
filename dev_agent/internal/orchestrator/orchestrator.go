package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	b "dev_agent/internal/brain"
	"dev_agent/internal/logx"

	t "dev_agent/internal/tools"
)

const systemPromptTemplate = `You are a expert software engineer, and a TDD (Test-Drive Development) workflow orchestrator.

### Agents
- **claude_code**: Analyze the requirement, Design and Implements solutions and tests. Summarizes work in '%[1]s/worklog.md'.
- **review_code**: Reviews code for P0/P1 issues. Records findings in '%[1]s/code_review.log'.

### Workflow
1.  **Prepare Codebase and Dev env (claude_code)**: When the task references a remote GitHub repository and the workspace '%[1]s' does not yet contain that codebase, clone it locally, install required dependencies, and record prep commands in '%[1]s/worklog.md'. Skip this step only when the workspace already has the necessary repo.
2.  **Implement (claude_code)**: Implement the solution and matching tests for the user's task.
3.  **Review (review_code)**: Review the implementation for P0/P1 issues.
4.  **Fix (claude_code)**: If issues are found, fix all P0/P1 issues and ensure tests pass.
5.  Repeat **Review** and **Fix** until 'review_code' agent reports no P0/P1 issues.

### Your Orchestration Rules
1.  **Call Agents**: For each workflow step, call 'execute_agent'.
2.  **Maintain State**: Track branch lineage ('parent_branch_id') and report any tool errors immediately.
3.  **Handle Review Data**: Before launching a **Fix** run, you **must** use 'read_artifact' to get the issues from 'code_review.log'.

### Agent Prompt Templates

Don't go into too much detail. You're just a TDD manager, clearly explain the tasks and let the agent analyze and execute them. So please Use the following prompt, Fill in the correct task and issues.
Never hard-code absolute filesystem paths; derive locations relative to the repository or the configured workspace root (%[1]s).

#### Preparation (claude_code)

You are preparing the workspace so downstream phases can analyze and ship code.

**User Task**: [The user's original task description - must be passed on exactly as is]

**Instructions**:

1.  **Detect Workspace State**
    * Inspect '%[1]s' for an existing repository (look for '.git' and expected project files).
    * If the repo already exists, document the finding in '%[1]s/worklog.md' and describe how to proceed; do not reclone.
2.  **Clone When Needed**
    * When the task references a remote GitHub repo and the workspace is missing it, clone the repo under '%[1]s'. Log the exact clone command in '%[1]s/worklog.md'.
    * Abort and log a "Preparation Failure" if the repo URL cannot be resolved.
3.  **Install Dependencies**
    * Immediately run the appropriate package/dependency installer(s) (e.g., npm install, pip install -r requirements.txt, go mod download) based on the manifests you discover.
    * Record every command and any notable output summaries in '%[1]s/worklog.md'.
4.  **Exit Criteria**
    * If cloning or dependency installation fails, stop and describe the failure in '%[1]s/worklog.md'—subsequent phases must not run.
    * When everything succeeds, summarize the prepared repo path, branch/default state, and installed tooling in '%[1]s/worklog.md'.

Ultrathink! Ensure the workspace is ready before implementation.
---

#### Implement (claude_code)

You are an expert engineer. Your goal is to produce high-quality, verified code based on deep analysis.

**User Task**: [The user's original task description - must be passed on exactly as is]

**Instructions**:

1.  **Phase 0: Context Verification (CRITICAL)**
    * Identify the issue or requirement metioned in the User Task (e.g., GitHub Issue IDs, specific requirement/error messages, requirement doc).
    * **Abort Condition**: If you cannot verify or locate the specific references (e.g., an Issue ID returns 404, or a mentioned file doesn't exist), you must **STOP IMMEDIATELY**.
        * Do not proceed to design or code.
        * Write a "Context Failure Report" to '%[1]s/worklog.md' explaining what was missing.
        * Inform the user that the task cannot be processed due to missing context.

	Hints: if needed, Use the 'gh' CLI to inspect GitHub issues/PRs just like 'git'; if either tool lacks auth, run '~/.setup-git.sh' to configure both before proceeding.


2.  **Phase 1: Analysis & Design** (Only if Phase 0 passes)
    * **Analyze**:
        * **For Bugs**: Perform Root Cause Analysis (RCA). Locate the code causing the issue.
        * **For Features**: Identify all code paths and files that need modification.
    * **Design**: Outline your solution strategy in '%[1]s/worklog.md'.

3.  **Phase 2: TDD Implementation**
    * **Test**: Write tests first. For bugs, ensure you have a regression test.
    * **Code**: Implement the solution according to your design.
    * **Verify**: Ensure local tests pass.

3.  **Final Step**: Update '%[1]s/worklog.md' with a summary of changes and test results.

Ultrathink! Analyze first, then code. Avoid over-engineering.
---

#### Review (review_code)

**User Task**: [The user's original task description]

**Instructions**:
1.  **Review Code Changes**: Review the recent modifications and tests to determine if they satisfy the User Task.
2.  **Scope**: Focus **ONLY** on the changed code and the direct impact of these changes.
    * **Do NOT** review unrelated legacy code or pre-existing issues unless they are made worse by this change.
3.  **Report**: Identify and log **P0 (Critical)** or **P1 (Major)** issues to '%[1]s/code_review.log'.
    * If the code meets the requirements and has no critical/major issues, report "No P0/P1 issues found".

Hints: if needed, Use the 'gh' CLI to inspect GitHub issues/PRs just like 'git'; if either tool lacks auth, run '~/.setup-git.sh' to configure both before proceeding.

Think it hard and

---

####  Fix (claude_code)

Ultrathink! Fix all P0/P1 issues reported in the review.

**Issues to Fix**:
[List of P0/P1 issues from '%[1]s/code_review.log']

**Original User Task**: [The user's original task description]

**Instructions**:
1.  **Address Issues**: Systematically fix every P0 and P1 issue listed.
2.  **Verify**: Ensure existing tests pass and add new tests if the review indicated missing coverage.
3.  **Update Log**: Append a "Fix Summary" to '%[1]s/worklog.md' explaining what was changed.

Hints: if needed, Use the 'gh' CLI to inspect GitHub issues/PRs just like 'git'; if either tool lacks auth, run '~/.setup-git.sh' to configure both before proceeding.

Ultrathink! Analyze first, then code. Avoid over-engineering.

### Completion
* Stop Condition: Stop when a review_code run reports no P0/P1 issues.
* Final Output: Reply with JSON only: {"is_finished": true, "task":"<original task>","summary":"<Concise outcome>"}
`

const (
	statusCompleted      = "completed"
	statusIterationLimit = "iteration_limit"

	iterationLimitSummary = "Reached iteration limit before clean review sign-off."
	defaultSuccessSummary = "Workflow completed successfully."
)

const maxIterations = 8

type publishHandler interface {
	BranchRange() map[string]string
	Handle(t.ToolCall) map[string]any
}

type PublishOptions struct {
	GitHubToken    string
	WorkspaceDir   string
	ParentBranchID string
	ProjectName    string
	Task           string
	GitUserName    string
	GitUserEmail   string
}

func finalizeBranchPush(handler publishHandler, opts PublishOptions, report map[string]any, success bool) (string, error) {
	if opts.GitHubToken == "" {
		return "", errors.New("missing GitHub token for publish step")
	}
	if strings.TrimSpace(opts.GitUserName) == "" {
		return "", errors.New("missing git user name for publish step")
	}
	if strings.TrimSpace(opts.GitUserEmail) == "" {
		return "", errors.New("missing git user email for publish step")
	}
	lineage := handler.BranchRange()
	parent := lineage["latest_branch_id"]
	if parent == "" {
		parent = opts.ParentBranchID
	}
	if parent == "" {
		return "", errors.New("unable to determine parent branch id for publish step")
	}

	outcome := iterationLimitSummary
	if success {
		summary := ""
		if report != nil {
			if s, ok := report["summary"].(string); ok && s != "" {
				summary = s
			}
		}
		if summary != "" {
			outcome = summary
		} else {
			outcome = defaultSuccessSummary
		}
	}

	meta := fmt.Sprintf("commit-meta: start_branch=%s latest_branch=%s", lineage["start_branch_id"], lineage["latest_branch_id"])
	tokenLiteral := strconv.Quote(opts.GitHubToken)
	identityInstruction := fmt.Sprintf("set git user.name to %q and user.email to %q (update both local and global config before committing).", opts.GitUserName, opts.GitUserEmail)
	prompt := fmt.Sprintf(`Finalize the task by committing and pushing the current workspace state.

Task: %[1]s
Outcome: %[2]s
GitHub access token (export for git auth and unset afterwards): %[3]s
Meta (include in the commit message if helpful): %[4]s

The worklog is located into '%[5]s/worklog.md'.

Choose an appropriate git branch name for this task, commit the related file changes, and reply with a concise publish report that MUST include: repository URL, pushed Git branch name, commit hash, and pointers to the latest implementation summary/tests (e.g., '%[5]s/worklog.md' and any test artifact). Do not print the raw token anywhere except when configuring git. If you cannot provide this report, treat the publish as failed.

Publishing rules:
- Configure git identity (%[6]s).
- Use the original user task and the latest entries in '%[5]s/worklog.md' to determine the target repository; confirm the repository root with 'git rev-parse --show-toplevel' and verify the remote via 'git remote -v'. Do not operate on an unrelated repo.
- Stage and commit only the files required for this task; exclude logs, review artifacts, and temporary scratch files.
- Keep branch names kebab-case and describe the task scope.
- Keep the commit subject <= 72 characters and meaningful.
- Unset exported credentials after pushing.
- Git push must be fully non-interactive. Rewrite the existing 'origin' remote to include the GitHub token (example: "CURRENT=$(git remote get-url origin); git remote set-url origin https://<github-username>:${GITHUB_TOKEN}@github.com/<owner>/<repo>.git"), run "git push -u origin <branch>", then restore the original remote URL. Do not print the raw token in logs.
- Do not stage or commit '%[5]s/worklog.md' or '%[5]s/code_review.log'.

Include a short publish report that states the repository URL, branch name, and a concise PR-style summary.`, opts.Task, outcome, tokenLiteral, meta, opts.WorkspaceDir, identityInstruction)

	logx.Infof("Finalizing workflow by asking claude_code to push from branch %s lineage.", parent)
	execArgs := map[string]any{
		"agent":            "claude_code",
		"prompt":           prompt,
		"parent_branch_id": parent,
	}
	if opts.ProjectName != "" {
		execArgs["project_name"] = opts.ProjectName
	}
	argsBytes, _ := json.Marshal(execArgs)
	execCall := t.ToolCall{Type: "function"}
	execCall.Function.Name = "execute_agent"
	execCall.Function.Arguments = string(argsBytes)

	execResp := handler.Handle(execCall)
	if status, _ := execResp["status"].(string); status != "success" {
		return "", fmt.Errorf("publish execute_agent failed: %v", execResp)
	}
	data, _ := execResp["data"].(map[string]any)
	branchID := t.ExtractBranchID(data)
	if branchID == "" {
		return "", errors.New("publish execute_agent missing branch id")
	}
	publishSummary := extractBranchOutput(data)
	if publishSummary == "" {
		logx.Warningf("Publish response missing required report (repo/branch/commit/tests); continuing without it (branch_id=%s)", branchID)
		publishSummary = fmt.Sprintf("Publish report unavailable; inspect Pantheon branch %s for push details.", branchID)
	}
	if report != nil {
		report["publish_report"] = publishSummary
		report["publish_pantheon_branch_id"] = branchID
	}
	if branchStatus := strings.TrimSpace(fmt.Sprintf("%v", data["status"])); branchStatus != "" {
		switch strings.ToLower(branchStatus) {
		case "failed":
			return "", fmt.Errorf("publish branch %s completed with failure status", branchID)
		}
	}

	return branchID, nil
}

func BuildInitialMessages(task, projectName, workspaceDir, parentBranchID string) []b.ChatMessage {
	systemPrompt := fmt.Sprintf(systemPromptTemplate, workspaceDir)
	userPayload := map[string]any{
		"task":             task,
		"parent_branch_id": parentBranchID,
		"project_name":     projectName,
		"workspace_dir":    workspaceDir,
		"notes":            "For every phase: craft an execute_agent prompt covering task, phase goal, context. Track branch lineage and stop when review_code reports no P0/P1 issues.",
	}
	content, _ := json.MarshalIndent(userPayload, "", "  ")
	return []b.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(content)},
	}
}

func assistantMessageToDict(msg b.ChatMessage) b.ChatMessage {
	// Already in the correct structure
	return msg
}

func ParseFinalReport(msg b.ChatMessage) (map[string]any, bool) {
	if msg.Content == "" {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(msg.Content), &m); err != nil {
		return nil, false
	}
	if m["is_finished"] == true {
		return m, true
	}
	return nil, false
}

func Orchestrate(brain *b.LLMBrain, handler *t.ToolHandler, messages []b.ChatMessage, publishOpts PublishOptions) (map[string]any, error) {
	tools := t.GetToolDefinitions()
	var (
		finalReport map[string]any
		finished    bool
		reviewCount int
	)

	for i := 1; ; i++ {
		logx.Infof("LLM iteration %d", i)
		resp, err := brain.Complete(messages, tools)
		if err != nil {
			return nil, err
		}
		choice := resp.Choices[0].Message
		messages = append(messages, assistantMessageToDict(choice))

		if len(choice.ToolCalls) > 0 {
			reviewCompleted := false
			for _, tc := range choice.ToolCalls {
				var args map[string]any
				if tc.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				htc := t.ToolCall{ID: tc.ID, Type: tc.Type}
				htc.Function.Name = tc.Function.Name
				htc.Function.Arguments = tc.Function.Arguments
				result := handler.Handle(htc)
				toolMsg := b.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: toJSON(result)}
				messages = append(messages, toolMsg)

				if tc.Function.Name == "execute_agent" {
					if agent, _ := args["agent"].(string); agent == "review_code" {
						if status, _ := result["status"].(string); status == "success" {
							reviewCompleted = true
						}
					}
				}
			}
			if reviewCompleted {
				reviewCount++
				logx.Infof("Completed review iteration %d/%d", reviewCount, maxIterations)
				if reviewCount >= maxIterations {
					logx.Errorf("Reached review iteration limit without final report.")
					break
				}
			}
			continue
		}

		if fr, ok := ParseFinalReport(choice); ok {
			finalReport = fr
			finished = true
			break
		}
		logx.Infof("Assistant response was not a final report; continuing.")
	}

	if finished {
		ensureReportDefaults(finalReport, publishOpts.Task, statusCompleted, true)
		_, err := finalizeBranchPush(handler, publishOpts, finalReport, true)
		if err != nil {
			return nil, err
		}
		return finalReport, nil
	}

	finalReport = map[string]any{
		"is_finished": false,
		"status":      statusIterationLimit,
		"task":        publishOpts.Task,
		"summary":     iterationLimitSummary,
	}
	branchID, err := finalizeBranchPush(handler, publishOpts, finalReport, false)
	if err != nil {
		return nil, err
	}
	if branchID != "" {
		logx.Infof("Workspace published to branch (branch_id=%s) after iteration limit.", branchID)
	}
	return finalReport, nil
}

func ChatLoop(brain *b.LLMBrain, handler *t.ToolHandler, messages []b.ChatMessage, maxIters int, publishOpts PublishOptions) (map[string]any, error) {
	if maxIters <= 0 {
		maxIters = maxIterations
	}
	tools := t.GetToolDefinitions()
	var (
		finalReport map[string]any
		finished    bool
		reviewCount int
	)

	for i := 1; ; i++ {
		fmt.Printf("[iter %d] requesting completion...\n", i)
		resp, err := brain.Complete(messages, tools)
		if err != nil {
			return nil, err
		}
		choice := resp.Choices[0].Message
		if choice.Content != "" {
			fmt.Printf("assistant> %s\n", choice.Content)
		}
		messages = append(messages, assistantMessageToDict(choice))

		if len(choice.ToolCalls) > 0 {
			reviewCompleted := false
			for _, tc := range choice.ToolCalls {
				fmt.Printf("tool> %s %s\n", tc.Function.Name, tc.Function.Arguments)
				var args map[string]any
				if tc.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				htc := t.ToolCall{ID: tc.ID, Type: tc.Type}
				htc.Function.Name = tc.Function.Name
				htc.Function.Arguments = tc.Function.Arguments
				result := handler.Handle(htc)
				js := toJSON(result)
				if len(js) > 2000 {
					js = js[:2000]
				}
				fmt.Printf("tool< %s\n", js)
				messages = append(messages, b.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: toJSON(result)})

				if tc.Function.Name == "execute_agent" {
					if agent, _ := args["agent"].(string); agent == "review_code" {
						if status, _ := result["status"].(string); status == "success" {
							reviewCompleted = true
						}
					}
				}
			}
			if reviewCompleted {
				reviewCount++
				fmt.Printf("note: completed review iteration %d/%d\n", reviewCount, maxIters)
				if reviewCount >= maxIters {
					logx.Errorf("Reached review iteration limit without final report.")
					break
				}
			}
			continue
		}
		if fr, ok := ParseFinalReport(choice); ok {
			finalReport = fr
			finished = true
			fmt.Println("assistant< final_report")
			break
		}
		fmt.Println("assistant< not final yet, continuing...")
	}

	if finished {
		ensureReportDefaults(finalReport, publishOpts.Task, statusCompleted, true)
		_, err := finalizeBranchPush(handler, publishOpts, finalReport, true)
		if err != nil {
			return nil, err
		}
		return finalReport, nil
	}

	finalReport = map[string]any{
		"is_finished": false,
		"status":      statusIterationLimit,
		"task":        publishOpts.Task,
		"summary":     iterationLimitSummary,
	}
	branchID, err := finalizeBranchPush(handler, publishOpts, finalReport, false)
	if err != nil {
		return nil, err
	}
	if branchID != "" {
		fmt.Fprintf(os.Stderr, "info: workspace pushed (branch_id=%s)\n", branchID)
	}
	return finalReport, nil
}

func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func ensureReportDefaults(report map[string]any, task, status string, finished bool) {
	if report == nil {
		return
	}
	if _, ok := report["task"]; !ok && task != "" {
		report["task"] = task
	}
	if _, ok := report["status"]; !ok && status != "" {
		report["status"] = status
	}
	if _, ok := report["is_finished"]; !ok {
		report["is_finished"] = finished
	}
}

func BuildInstructions(report map[string]any) string {
	if report == nil {
		return ""
	}
	start := reportString(report, "start_branch_id")
	latest := reportString(report, "latest_branch_id")
	status := reportString(report, "status")
	publishReport := reportString(report, "publish_report")
	publishBranch := reportString(report, "publish_pantheon_branch_id")

	var parts []string

	switch {
	case start != "" && latest != "":
		if start == latest {
			parts = append(parts, fmt.Sprintf("Branch lineage: start=%s, latest=%s. Inspect manifest %s in Pantheon to review artifacts.", start, latest, latest))
		} else {
			parts = append(parts, fmt.Sprintf("Branch lineage: start=%s → latest=%s. Inspect manifest %s in Pantheon to review artifacts.", start, latest, latest))
		}
	case latest != "":
		parts = append(parts, fmt.Sprintf("Inspect manifest %s in Pantheon to review artifacts.", latest))
	case start != "":
		parts = append(parts, fmt.Sprintf("Branch lineage started from %s; inspect it in Pantheon to review artifacts.", start))
	}

	if publishReport != "" {
		parts = append(parts, fmt.Sprintf("Publish report describes the GitHub push target: %s", publishReport))
	}

	if publishBranch != "" {
		parts = append(parts, fmt.Sprintf("Github Push from pantheon branch: %s.", publishBranch))
	}

	switch status {
	case statusIterationLimit:
		target := latest
		if target == "" {
			target = start
		}
		if target != "" {
			parts = append(parts, fmt.Sprintf("Next (if your are allowed or instructed), you can rerun dev-agent with --parent-branch-id %s to continue automated iterations;", target))
		}
	default:
		if publishReport != "" {
			parts = append(parts, "Next step: review the pushed GitHub branch and, based on your process, proceed with the normal PR/merge workflow.")
		} else if latest != "" {
			parts = append(parts, "Next step: review the manifest and test results above, then proceed with whichever merge/publish flow fits your process.")
		}
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

func reportString(report map[string]any, key string) string {
	if report == nil {
		return ""
	}
	if v, ok := report[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}

func extractBranchOutput(data map[string]any) string {
	if data == nil {
		return ""
	}
	branch, _ := data["branch"].(map[string]any)
	if branch == nil {
		return ""
	}
	if out, _ := branch["output"].(string); strings.TrimSpace(out) != "" {
		return strings.TrimSpace(out)
	}
	if snap, _ := branch["latest_snap"].(map[string]any); snap != nil {
		if out, _ := snap["output"].(string); strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out)
		}
	}
	if manifest, _ := branch["manifest"].(map[string]any); manifest != nil {
		if summary, _ := manifest["summary"].(string); strings.TrimSpace(summary) != "" {
			return strings.TrimSpace(summary)
		}
	}
	return ""
}
