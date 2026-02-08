package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	b "dev_agent_v2/internal/brain"
	cfg "dev_agent_v2/internal/config"
	"dev_agent_v2/internal/logx"
	o "dev_agent_v2/internal/orchestrator"
	"dev_agent_v2/internal/streaming"
	t "dev_agent_v2/internal/tools"
)

func main() {
	task := flag.String("task", "", "Task description (should include the issue link or existing PR link)")
	parent := flag.String("parent-branch-id", "", "Parent branch UUID (required)")
	project := flag.String("project-name", "", "Optional project name override")
	headless := flag.Bool("headless", false, "Run in headless mode (no chat prints)")
	streamJSON := flag.Bool("stream-json", false, "Emit orchestration events as NDJSON to stdout (forces headless mode)")
	maxTurns := flag.Int("max-turns", 0, "Maximum LLM turns before stopping (0 uses default)")
	flag.Parse()

	streamEnabled := streamJSON != nil && *streamJSON
	if streamEnabled {
		*headless = true
		logx.SetLevel(logx.Error)
	}

	conf, err := cfg.FromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	if *project != "" {
		conf.ProjectName = *project
	}
	if strings.TrimSpace(conf.ProjectName) == "" {
		fmt.Fprintln(os.Stderr, "Project name must be provided via PROJECT_NAME or --project-name")
		os.Exit(1)
	}
	if strings.TrimSpace(*parent) == "" {
		fmt.Fprintln(os.Stderr, "--parent-branch-id is required")
		os.Exit(1)
	}

	tsk := strings.TrimSpace(*task)
	if tsk == "" {
		promptWriter := os.Stdout
		if streamEnabled {
			promptWriter = os.Stderr
		}
		fmt.Fprintf(promptWriter, "you> Enter task description: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		tsk = strings.TrimSpace(line)
		if tsk == "" {
			fmt.Fprintln(os.Stderr, "error: --task is required")
			os.Exit(1)
		}
	}

	brain := b.NewLLMBrain(conf.AzureAPIKey, conf.AzureEndpoint, conf.AzureDeployment, conf.AzureAPIVersion, 3)
	mcp := t.NewMCPClient(conf.MCPBaseURL)
	handler := t.NewToolHandler(mcp, conf.ProjectName, *parent, conf.WorkspaceDir, &t.ToolHandlerTiming{
		PollTimeout: conf.PollTimeout,
		PollInitial: conf.PollInitial,
		PollMax:     conf.PollMax,
		PollBackoff: conf.PollBackoffFactor,
	})

	msgs := o.BuildInitialMessages(tsk, conf.ProjectName, conf.WorkspaceDir, *parent)

	var streamer *streaming.JSONStreamer
	if streamEnabled {
		streamer = streaming.NewJSONStreamer(true, os.Stdout)
		streamer.EmitThreadStarted(tsk, conf.ProjectName, *parent, *headless)
	}

	opts := o.RunOptions{
		Task:     tsk,
		Streamer: streamer,
		MaxTurns: *maxTurns,
	}

	var report map[string]any
	if *headless {
		report, err = o.Orchestrate(brain, handler, msgs, opts)
	} else {
		report, err = o.ChatLoop(brain, handler, msgs, 0, opts)
	}
	if err != nil {
		if streamer != nil && streamer.Enabled() {
			streamer.EmitError("cli", err.Error(), nil)
			streamer.EmitThreadCompleted("error", err.Error(), nil)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Attach observed branch range and instructions
	br := handler.BranchRange()
	if report == nil {
		report = map[string]any{}
	}
	if start, ok := br["start_branch_id"]; ok {
		report["start_branch_id"] = start
	}
	if latest, ok := br["latest_branch_id"]; ok {
		report["latest_branch_id"] = latest
	}
	if _, ok := report["task"]; !ok {
		report["task"] = tsk
	}
	sanitizeFinalReport(report)
	if finalized, ferr := finalizeReportWithBrain(brain, report); ferr == nil && finalized != nil {
		report = finalized
	}
	sanitizeFinalReport(report)

	if streamer != nil && streamer.Enabled() {
		status, _ := report["status"].(string)
		summary, _ := report["summary"].(string)
		streamer.EmitThreadCompleted(status, summary, report)
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Fprintln(os.Stderr, string(out))
}

func sanitizeFinalReport(report map[string]any) {
	if report == nil {
		return
	}
	allowed := map[string]struct{}{
		"is_finished":      {},
		"status":           {},
		"summary":          {},
		"instructions":     {},
		"task":             {},
		"start_branch_id":  {},
		"latest_branch_id": {},
		"pr_url":           {},
		"pr_number":        {},
		"pr_head_branch":   {},
		"error":            {},
		"instruction":      {}, // optional; used by FINISHED_WITH_ERROR flows
	}
	for k := range report {
		if _, ok := allowed[k]; !ok {
			delete(report, k)
		}
	}

	// Drop empty PR fields to keep the JSON clean.
	dropNilOrEmptyString(report, "pr_url")
	dropNilOrEmptyString(report, "pr_head_branch")
	dropNilOrEmptyString(report, "instructions")

	if v, ok := report["pr_number"]; !ok || v == nil || isZeroNumber(v) {
		delete(report, "pr_number")
	}
}

func dropNilOrEmptyString(m map[string]any, key string) {
	if m == nil {
		return
	}
	v, ok := m[key]
	if !ok || v == nil {
		delete(m, key)
		return
	}
	if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
		delete(m, key)
	}
}

func isZeroNumber(v any) bool {
	switch n := v.(type) {
	case int:
		return n == 0
	case int64:
		return n == 0
	case float64:
		return int64(n) == 0
	case json.Number:
		i, err := n.Int64()
		return err == nil && i == 0
	case string:
		s := strings.TrimSpace(n)
		return s == "" || s == "0"
	default:
		return false
	}
}

func finalizeReportWithBrain(brain *b.LLMBrain, report map[string]any) (map[string]any, error) {
	if brain == nil || report == nil {
		return nil, fmt.Errorf("missing brain or report")
	}

	input := map[string]any{}
	for k, v := range report {
		input[k] = v
	}

	inBytes, _ := json.MarshalIndent(input, "", "  ")

	system := `You are the dev-agent-v2 report finalizer.

Given an input report from an automated Pantheon run, output a CLEAN final JSON report.

Rules:
- Output JSON only. No code fences, no prose.
- Keep the original status semantics; do not invent success/failure.
- Required fields in output: is_finished=true, status, task, summary, instructions.
- Allowed optional fields (only if present and non-empty): start_branch_id, latest_branch_id, pr_url, pr_number, pr_head_branch, error, instruction.
- Do NOT include any other keys.
- Do NOT include any null/empty fields.

instructions should be actionable next steps. If status is iteration_limit, include rerun guidance using latest_branch_id as --parent-branch-id.`

	user := fmt.Sprintf("INPUT_REPORT_JSON:\n%s", string(inBytes))
	msgs := []b.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
	resp, err := brain.Complete(msgs, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("finalizer returned empty response")
	}
	text := strings.TrimSpace(resp.Choices[0].Message.Content)
	out, err := parseJSONObject(text)
	if err != nil {
		return nil, err
	}

	// Fill required fields from the input when missing.
	out["is_finished"] = true
	if _, ok := out["status"]; !ok {
		out["status"] = report["status"]
	}
	if _, ok := out["task"]; !ok {
		out["task"] = report["task"]
	}
	if _, ok := out["start_branch_id"]; !ok {
		if v, ok := report["start_branch_id"]; ok {
			out["start_branch_id"] = v
		}
	}
	if _, ok := out["latest_branch_id"]; !ok {
		if v, ok := report["latest_branch_id"]; ok {
			out["latest_branch_id"] = v
		}
	}
	if _, ok := out["pr_url"]; !ok {
		if v, ok := report["pr_url"]; ok {
			out["pr_url"] = v
		}
	}
	if _, ok := out["pr_number"]; !ok {
		if v, ok := report["pr_number"]; ok {
			out["pr_number"] = v
		}
	}
	if _, ok := out["pr_head_branch"]; !ok {
		if v, ok := report["pr_head_branch"]; ok {
			out["pr_head_branch"] = v
		}
	}
	if _, ok := out["error"]; !ok {
		if v, ok := report["error"]; ok {
			out["error"] = v
		}
	}

	if strings.TrimSpace(fmt.Sprintf("%v", out["summary"])) == "" {
		return nil, fmt.Errorf("finalizer output missing summary")
	}
	if strings.TrimSpace(fmt.Sprintf("%v", out["instructions"])) == "" {
		return nil, fmt.Errorf("finalizer output missing instructions")
	}

	return out, nil
}

func parseJSONObject(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty JSON content")
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err == nil {
		return obj, nil
	}

	// Best-effort: extract the first JSON object from a response that includes extra text.
	start := strings.IndexAny(text, "{")
	if start < 0 {
		return nil, fmt.Errorf("no JSON object found")
	}
	dec := json.NewDecoder(strings.NewReader(text[start:]))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	return obj, nil
}
