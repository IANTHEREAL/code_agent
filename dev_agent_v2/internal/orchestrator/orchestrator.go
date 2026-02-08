package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	b "dev_agent_v2/internal/brain"
	"dev_agent_v2/internal/logx"
	"dev_agent_v2/internal/streaming"

	t "dev_agent_v2/internal/tools"
)

const systemPromptPreamble = `You are an expert software engineer and a Pantheon workflow orchestrator.

You control a strict, evidence-first Issue/PR resolve loop by launching long-running Pantheon branches via tools.

Hard rule: each assistant response MUST either (a) call exactly ONE tool, or (b) output the FINAL REPORT as JSON (and nothing else).
Hard rule: one issue, one PR. Never create a second PR.
Hard rule: num_branches is always 1 (execute_agent enforces this).
Hard rule: no publish step (do not add any final publish stage outside explorations).

Tool note: execute_agent returns a response excerpt (may be truncated). If response_truncated=true, use full_output_hint to fetch more via branch_output with tail/max_chars.
`

func loadPlaybookFromDisk() string {
	// Try a few common locations depending on where the CLI is invoked from.
	candidates := []string{
		filepath.Join("dev_agent_v2", "SKILL.md"),
		"SKILL.md",
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(data))
		if text != "" {
			return text
		}
	}
	return ""
}

func buildSystemPrompt() string {
	var sb strings.Builder
	sb.WriteString(systemPromptPreamble)

	if playbook := loadPlaybookFromDisk(); playbook != "" {
		sb.WriteString("\n\n=== Playbook (authoritative) ===\n\n")
		sb.WriteString(playbook)
	}
	sb.WriteString("\n\n=== Output rule ===\n\n")
	sb.WriteString("Stop by outputting JSON only, matching the playbook’s final output shape. Never include any extra text around the JSON.\n")
	return sb.String()
}

const (
	statusCompleted         = "completed"
	statusIterationLimit    = "iteration_limit"
	statusFinishedWithError = "FINISHED_WITH_ERROR"

	iterationLimitSummary = "Reached iteration limit before clean review sign-off."
)

const (
	defaultMaxTurns      = 60
	defaultMaxToolCalls  = 40
	defaultMaxRetryTurns = 6
)

type RunOptions struct {
	Task     string
	Streamer *streaming.JSONStreamer
	MaxTurns int
}

func BuildInitialMessages(task, projectName, workspaceDir, parentBranchID string) []b.ChatMessage {
	systemPrompt := buildSystemPrompt()
	userPayload := map[string]any{
		"task":             strings.TrimSpace(task),
		"parent_branch_id": parentBranchID,
		"project_name":     projectName,
		"workspace_dir":    workspaceDir,
		"notes":            "Follow the playbook in the system prompt. One tool call per turn. One PR only. No publish step.",
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

func Orchestrate(brain *b.LLMBrain, handler *t.ToolHandler, messages []b.ChatMessage, opts RunOptions) (map[string]any, error) {
	tools := t.GetToolDefinitions()
	emitter := newEventEmitter(opts.Streamer)
	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}
	var (
		finalReport           map[string]any
		finished              bool
		errorState            bool
		totalToolCalls        int
		executedToolCalls     int
		consecutiveRetryTurns int
	)

	for i := 1; i <= maxTurns; i++ {
		logx.Infof("LLM iteration %d", i)
		turnID := fmt.Sprintf("turn_%d", i)
		if emitter != nil {
			emitter.TurnStarted(turnID, i, len(messages), totalToolCalls)
		}
		resp, err := brain.Complete(messages, tools)
		if err != nil {
			if emitter != nil {
				emitter.EmitError("llm.complete", err.Error(), map[string]any{"iteration": i, "turn_id": turnID})
			}
			return nil, err
		}
		choice := resp.Choices[0].Message
		messages = append(messages, assistantMessageToDict(choice))
		if emitter != nil {
			emitter.AssistantMessage(turnID, choice.Content, len(choice.ToolCalls))
		}

		if len(choice.ToolCalls) > 0 {
			turnToolCount := len(choice.ToolCalls)
			totalToolCalls += turnToolCount

			// Enforce "single tool call per turn" in code (not just in the system prompt).
			if turnToolCount != 1 {
				msg := fmt.Sprintf("policy violation: expected exactly 1 tool call, got %d. No tool calls executed. Please retry with a single tool call.", turnToolCount)
				if emitter != nil {
					emitter.EmitError("tool_policy", msg, map[string]any{"tool_call_count": turnToolCount, "turn_id": turnID})
				}
				consecutiveRetryTurns++
				for _, tc := range choice.ToolCalls {
					synth := map[string]any{
						"status": "error",
						"error": map[string]any{
							"message": msg,
						},
					}
					toolMsg := b.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: toJSON(synth)}
					messages = append(messages, toolMsg)
				}
				if emitter != nil {
					emitter.TurnCompleted(turnID, i, turnToolCount, false)
				}
				if consecutiveRetryTurns >= defaultMaxRetryTurns {
					finalReport = buildErrorFinalReport(opts.Task, "Too many policy-violation retries; aborting.", "", map[string]any{"turn_id": turnID})
					finished = true
					errorState = true
					break
				}
				continue
			}

			consecutiveRetryTurns = 0
			tc := choice.ToolCalls[0]
			args := parseToolArgs(tc.Function.Arguments)

			var itemArgs map[string]any
			if emitter != nil {
				itemArgs = sanitizeToolArgs(tc.Function.Name, args)
			}
			itemID := ""
			if emitter != nil {
				itemID = emitter.ItemStarted("tool_call", tc.Function.Name, itemArgs)
			}
			htc := t.ToolCall{ID: tc.ID, Type: tc.Type}
			htc.Function.Name = tc.Function.Name
			htc.Function.Arguments = tc.Function.Arguments
			var start time.Time
			if emitter != nil {
				start = time.Now()
			}
			result := handler.Handle(htc)
			executedToolCalls++
			var duration time.Duration
			if emitter != nil {
				duration = time.Since(start)
			}
			toolMsg := b.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: toJSON(result)}
			messages = append(messages, toolMsg)
			if emitter != nil {
				emitter.ItemCompleted(itemID, resultStatus(result), duration, eventBranchID(result), summarizeToolResult(result))
			}

			if instr, summaryMsg, details := toolInstruction(result); instr != "" {
				if emitter != nil {
					emitter.EmitError("tool_instruction", summaryMsg, map[string]any{"instruction": instr})
				}
				finalReport = buildErrorFinalReport(opts.Task, summaryMsg, instr, details)
				finished = true
				errorState = true
				if emitter != nil {
					emitter.TurnCompleted(turnID, i, turnToolCount, false)
				}
				break
			}
			if emitter != nil {
				emitter.TurnCompleted(turnID, i, turnToolCount, false)
			}
			if executedToolCalls >= defaultMaxToolCalls {
				logx.Errorf("Reached tool-call limit without final report.")
				break
			}
			continue
		}

		hasFinal := false
		if fr, ok := ParseFinalReport(choice); ok {
			finalReport = fr
			finished = true
			hasFinal = true
		} else {
			logx.Infof("Assistant response was not a final report; continuing.")
		}
		if emitter != nil {
			emitter.TurnCompleted(turnID, i, 0, hasFinal)
		}
		if finished {
			break
		}
	}

	if finished {
		if errorState {
			ensureReportDefaults(finalReport, opts.Task, statusFinishedWithError, true)
			return finalReport, nil
		}
		ensureReportDefaults(finalReport, opts.Task, statusCompleted, true)
		return finalReport, nil
	}

	finalReport = map[string]any{
		"is_finished": false,
		"status":      statusIterationLimit,
		"task":        opts.Task,
		"summary":     iterationLimitSummary,
	}
	return finalReport, nil
}

func ChatLoop(brain *b.LLMBrain, handler *t.ToolHandler, messages []b.ChatMessage, maxIters int, opts RunOptions) (map[string]any, error) {
	if maxIters <= 0 {
		maxIters = defaultMaxTurns
	}
	tools := t.GetToolDefinitions()
	var (
		finalReport map[string]any
		finished    bool
		errorState  bool
	)

	for i := 1; i <= maxIters; i++ {
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
			stopDueToInstruction := false
			if len(choice.ToolCalls) != 1 {
				msg := fmt.Sprintf("policy violation: expected exactly 1 tool call, got %d. No tool calls executed. Please retry with a single tool call.", len(choice.ToolCalls))
				fmt.Printf("note: %s\n", msg)
				for _, tc := range choice.ToolCalls {
					synth := map[string]any{"status": "error", "error": map[string]any{"message": msg}}
					messages = append(messages, b.ChatMessage{Role: "tool", ToolCallID: tc.ID, Content: toJSON(synth)})
				}
				continue
			}

			tc := choice.ToolCalls[0]
			fmt.Printf("tool> %s %s\n", tc.Function.Name, tc.Function.Arguments)
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

			if instr, summaryMsg, details := toolInstruction(result); instr != "" {
				finalReport = buildErrorFinalReport(opts.Task, summaryMsg, instr, details)
				finished = true
				errorState = true
				stopDueToInstruction = true
			}
			if stopDueToInstruction {
				break
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
		if errorState {
			ensureReportDefaults(finalReport, opts.Task, statusFinishedWithError, true)
			return finalReport, nil
		}
		ensureReportDefaults(finalReport, opts.Task, statusCompleted, true)
		return finalReport, nil
	}

	finalReport = map[string]any{
		"is_finished": false,
		"status":      statusIterationLimit,
		"task":        opts.Task,
		"summary":     iterationLimitSummary,
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

type eventEmitter struct {
	streamer *streaming.JSONStreamer
	nextItem int
}

func newEventEmitter(streamer *streaming.JSONStreamer) *eventEmitter {
	if streamer == nil || !streamer.Enabled() {
		return nil
	}
	return &eventEmitter{streamer: streamer}
}

func (e *eventEmitter) TurnStarted(turnID string, iteration, messageCount, toolCount int) {
	if e == nil {
		return
	}
	e.streamer.EmitTurnStarted(turnID, iteration, messageCount, toolCount)
}

func (e *eventEmitter) AssistantMessage(turnID, preview string, toolCalls int) {
	if e == nil {
		return
	}
	e.streamer.EmitAssistantMessage(turnID, preview, toolCalls)
}

func (e *eventEmitter) TurnCompleted(turnID string, iteration, toolCalls int, hasFinal bool) {
	if e == nil {
		return
	}
	e.streamer.EmitTurnCompleted(turnID, iteration, toolCalls, hasFinal)
}

func (e *eventEmitter) ItemStarted(kind, name string, args map[string]any) string {
	if e == nil {
		return ""
	}
	e.nextItem++
	itemID := fmt.Sprintf("item_%d", e.nextItem)
	e.streamer.EmitItemStarted(itemID, kind, name, args)
	return itemID
}

func (e *eventEmitter) ItemCompleted(itemID, status string, duration time.Duration, branchID, summary string) {
	if e == nil || itemID == "" {
		return
	}
	e.streamer.EmitItemCompleted(itemID, status, duration, branchID, summary)
}

func (e *eventEmitter) EmitError(scope, message string, extra map[string]any) {
	if e == nil {
		return
	}
	e.streamer.EmitError(scope, message, extra)
}

func parseToolArgs(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return map[string]any{}
	}
	return args
}

func sanitizeToolArgs(name string, args map[string]any) map[string]any {
	if len(args) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	switch name {
	case "execute_agent":
		copyStringField(out, args, "agent")
		copyStringField(out, args, "project_name")
		copyStringField(out, args, "parent_branch_id")
		if prompt, _ := args["prompt"].(string); prompt != "" {
			preview := streaming.PromptPreview(prompt)
			out["prompt_preview"] = preview
			if strings.TrimSpace(prompt) != preview {
				out["prompt_truncated"] = true
			}
		}
	case "read_artifact":
		copyStringField(out, args, "branch_id")
		copyStringField(out, args, "path")
	case "branch_output":
		copyStringField(out, args, "branch_id")
		copyBoolField(out, args, "full_output")
	default:
		for k, v := range args {
			switch val := v.(type) {
			case string:
				out[k] = streaming.PromptPreview(val)
			case float64, bool:
				out[k] = val
			}
		}
	}
	return out
}

func copyStringField(dst, src map[string]any, key string) {
	if val, ok := src[key].(string); ok && strings.TrimSpace(val) != "" {
		dst[key] = strings.TrimSpace(val)
	}
}

func copyFloatField(dst, src map[string]any, key string) {
	if val, ok := src[key].(float64); ok {
		dst[key] = val
	}
}

func copyBoolField(dst, src map[string]any, key string) {
	if val, ok := src[key].(bool); ok {
		dst[key] = val
	}
}

func resultStatus(resp map[string]any) string {
	if resp == nil {
		return "error"
	}
	if status, ok := resp["status"].(string); ok && strings.TrimSpace(status) != "" {
		return strings.ToLower(strings.TrimSpace(status))
	}
	return "error"
}

func eventBranchID(resp map[string]any) string {
	if resp == nil {
		return ""
	}
	if data, _ := resp["data"].(map[string]any); data != nil {
		if id := t.ExtractBranchID(data); id != "" {
			return id
		}
	}
	return t.ExtractBranchID(resp)
}

func summarizeToolResult(resp map[string]any) string {
	if resp == nil {
		return ""
	}
	if errMsg, _ := resp["error"].(string); strings.TrimSpace(errMsg) != "" {
		return streaming.PromptPreview(errMsg)
	}
	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		if msg, _ := errObj["message"].(string); strings.TrimSpace(msg) != "" {
			return streaming.PromptPreview(msg)
		}
		if instr, _ := errObj["instruction"].(string); strings.TrimSpace(instr) != "" {
			return fmt.Sprintf("instruction=%s", strings.TrimSpace(instr))
		}
	}
	if data, _ := resp["data"].(map[string]any); data != nil {
		if out, _ := data["response"].(string); strings.TrimSpace(out) != "" {
			return streaming.PromptPreview(out)
		}
		if status, _ := data["status"].(string); strings.TrimSpace(status) != "" {
			return fmt.Sprintf("status=%s", strings.TrimSpace(status))
		}
	}
	if status, _ := resp["status"].(string); strings.TrimSpace(status) != "" {
		return fmt.Sprintf("status=%s", strings.TrimSpace(status))
	}
	return ""
}

func toolInstruction(resp map[string]any) (string, string, map[string]any) {
	if resp == nil {
		return "", "", nil
	}
	if errObj, ok := resp["error"].(map[string]any); ok {
		instr := strings.TrimSpace(reportString(errObj, "instruction"))
		message := strings.TrimSpace(reportString(errObj, "message"))
		var details map[string]any
		if det, ok := errObj["details"].(map[string]any); ok && len(det) > 0 {
			details = det
		}
		return instr, message, details
	}
	if msg, ok := resp["error"].(string); ok {
		return "", strings.TrimSpace(msg), nil
	}
	return "", "", nil
}

func buildErrorFinalReport(task, summary, instruction string, details map[string]any) map[string]any {
	out := map[string]any{
		"is_finished": true,
		"status":      statusFinishedWithError,
	}
	if strings.TrimSpace(task) != "" {
		out["task"] = strings.TrimSpace(task)
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "Workflow halted due to a tool execution error."
	}
	out["summary"] = summary
	if instruction != "" {
		out["instruction"] = instruction
	}
	errPayload := map[string]any{}
	if summary != "" {
		errPayload["message"] = summary
	}
	if instruction != "" {
		errPayload["instruction"] = instruction
	}
	if len(details) > 0 {
		errPayload["details"] = details
	}
	if len(errPayload) > 0 {
		out["error"] = errPayload
	}
	return out
}
