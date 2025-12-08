package tools

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestExecuteAgentReviewCodeRetriesMissingLog(t *testing.T) {
	client := &fakeMCPClient{
		readResults: []branchReadResult{
			{data: map[string]any{"error": "404: File or directory not found: /workspace/code_review.log"}},
			{err: notFoundErr(2)},
			{data: map[string]any{"content": "ok"}},
		},
	}
	handler := &ToolHandler{
		client:        client,
		defaultProj:   "proj",
		branchTracker: NewBranchTracker("parent"),
		workspaceDir:  "/workspace",
	}

	args := map[string]any{
		"agent":            "review_code",
		"prompt":           "review the latest changes",
		"parent_branch_id": "parent",
		"project_name":     "proj",
	}

	res, err := handler.executeAgent(args)
	if err != nil {
		t.Fatalf("executeAgent returned error: %v", err)
	}

	if got := client.parallelExploreCalls; got != 3 {
		t.Fatalf("expected 3 execute attempts, got %d", got)
	}
	if got := len(client.branchReadInputs); got != 3 {
		t.Fatalf("expected 3 read_artifact attempts, got %d", got)
	}
	for idx, input := range client.branchReadInputs {
		if input.path != "/workspace/code_review.log" {
			t.Fatalf("read attempt %d used path %q", idx+1, input.path)
		}
	}
	if got := res["branch_id"]; got != "branch-3" {
		t.Fatalf("expected final branch_id branch-3, got %#v", got)
	}
	if report, ok := res["review_report"].(string); !ok || strings.TrimSpace(report) != "ok" {
		t.Fatalf("expected review_report=ok, got %#v", res["review_report"])
	}
}

func TestExecuteAgentReviewCodeFailsAfterMaxAttempts(t *testing.T) {
	client := &fakeMCPClient{
		readResults: []branchReadResult{
			{err: notFoundErr(1)},
			{err: notFoundErr(2)},
			{err: notFoundErr(3)},
		},
	}
	handler := &ToolHandler{
		client:        client,
		defaultProj:   "proj",
		branchTracker: NewBranchTracker("parent"),
		workspaceDir:  "/workspace",
	}

	args := map[string]any{
		"agent":            "review_code",
		"prompt":           "review the latest changes",
		"parent_branch_id": "parent",
		"project_name":     "proj",
	}

	_, err := handler.executeAgent(args)
	if err == nil {
		t.Fatalf("expected error after max attempts, got nil")
	}

	var te ToolExecutionError
	if !errors.As(err, &te) {
		t.Fatalf("expected ToolExecutionError, got %T", err)
	}
	if te.Instruction != "FINISHED_WITH_ERROR" {
		t.Fatalf("expected FINISHED_WITH_ERROR instruction, got %q", te.Instruction)
	}
	if te.Details["attempts"] != 3 {
		t.Fatalf("expected attempts=3 in details, got %#v", te.Details["attempts"])
	}
}

func TestHandleBranchOutputRequiresBranchID(t *testing.T) {
	handler := &ToolHandler{
		client:        &fakeMCPClient{},
		branchTracker: NewBranchTracker("parent"),
	}
	call := ToolCall{}
	call.Function.Name = "branch_output"
	call.Function.Arguments = "{}"

	res := handler.Handle(call)
	if status := res["status"]; status != "error" {
		t.Fatalf("expected status error, got %#v", status)
	}
	errPayload, _ := res["error"].(map[string]any)
	if errPayload["message"] != "`branch_id` is required" {
		t.Fatalf("expected missing branch_id message, got %#v", errPayload["message"])
	}
}

func TestHandleBranchOutputCallsClient(t *testing.T) {
	client := &fakeMCPClient{
		branchOutputResult: map[string]any{"output": "short"},
	}
	handler := &ToolHandler{
		client:        client,
		branchTracker: NewBranchTracker("parent"),
	}
	call := ToolCall{}
	call.Function.Name = "branch_output"
	call.Function.Arguments = `{"branch_id":"branch-123","full_output":true}`

	res := handler.Handle(call)
	if status := res["status"]; status != "success" {
		t.Fatalf("expected status success, got %#v", status)
	}
	data, _ := res["data"].(map[string]any)
	if data["output"] != "short" {
		t.Fatalf("unexpected data payload %#v", data)
	}
	if len(client.branchOutputInputs) != 1 {
		t.Fatalf("expected 1 branch_output call, got %d", len(client.branchOutputInputs))
	}
	if got := client.branchOutputInputs[0]; got.branchID != "branch-123" || !got.fullOutput {
		t.Fatalf("unexpected branch_output args: %#v", got)
	}
}

func TestHandleBranchOutputDefaultsFullOutputFalse(t *testing.T) {
	client := &fakeMCPClient{
		branchOutputResult: map[string]any{"output": "partial"},
	}
	handler := &ToolHandler{
		client:        client,
		branchTracker: NewBranchTracker("parent"),
	}
	call := ToolCall{}
	call.Function.Name = "branch_output"
	call.Function.Arguments = `{"branch_id":"branch-234"}`

	_ = handler.Handle(call)
	if len(client.branchOutputInputs) != 1 {
		t.Fatalf("expected 1 branch_output call, got %d", len(client.branchOutputInputs))
	}
	if got := client.branchOutputInputs[0]; got.fullOutput {
		t.Fatalf("expected default full_output=false, got true")
	}
}

func TestReadArtifactHandlesErrorPayload(t *testing.T) {
	client := &fakeMCPClient{
		readResults: []branchReadResult{
			{data: map[string]any{"error": "404: File or directory not found: /workspace/missing.log"}},
		},
	}
	handler := &ToolHandler{
		client:        client,
		branchTracker: NewBranchTracker("parent"),
	}
	call := ToolCall{}
	call.Function.Name = "read_artifact"
	call.Function.Arguments = `{"branch_id":"branch-1","path":"/workspace/missing.log"}`

	res := handler.Handle(call)
	if status := res["status"]; status != "error" {
		t.Fatalf("expected status error, got %#v", status)
	}
	errMsg, _ := res["error"].(string)
	if !strings.Contains(errMsg, "404") || !strings.Contains(errMsg, "missing.log") {
		t.Fatalf("unexpected error message %#v", errMsg)
	}
}

func TestCheckStatusUsesConfiguredTimeout(t *testing.T) {
	attempts := 0
	client := &fakeMCPClient{
		getBranchFunc: func(branchID string) (map[string]any, error) {
			attempts++
			return map[string]any{
				"id":             branchID,
				"status":         "running",
				"latest_snap_id": fmt.Sprintf("snap-%d", attempts),
			}, nil
		},
	}
	clock := newFakeClock()
	handler := &ToolHandler{
		client:              client,
		branchTracker:       NewBranchTracker("parent"),
		branchStatusTimeout: 5 * time.Second,
		clock:               clock,
	}

	_, err := handler.checkStatus(map[string]any{
		"branch_id":                 "branch-1",
		"poll_interval_seconds":     1,
		"max_poll_interval_seconds": 1,
	})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	var te ToolExecutionError
	if !errors.As(err, &te) || !strings.Contains(strings.ToLower(te.Msg), "timed out") {
		t.Fatalf("expected timeout ToolExecutionError, got %#v", err)
	}
	elapsed := clock.Elapsed()
	if elapsed < handler.branchStatusTimeout {
		t.Fatalf("expected to wait at least %s, got %s", handler.branchStatusTimeout, elapsed)
	}
	if elapsed > handler.branchStatusTimeout*2 {
		t.Fatalf("expected timeout near %s, got %s", handler.branchStatusTimeout, elapsed)
	}
}

func TestCheckStatusAllowsSlowCompletionWithinTimeout(t *testing.T) {
	attempts := 0
	client := &fakeMCPClient{
		getBranchFunc: func(branchID string) (map[string]any, error) {
			attempts++
			status := "running"
			if attempts >= 3 {
				status = "succeed"
			}
			return map[string]any{
				"id":             branchID,
				"status":         status,
				"latest_snap_id": fmt.Sprintf("snap-%d", attempts),
			}, nil
		},
	}
	clock := newFakeClock()
	handler := &ToolHandler{
		client:              client,
		branchTracker:       NewBranchTracker("parent"),
		branchStatusTimeout: 10 * time.Second,
		clock:               clock,
	}

	resp, err := handler.checkStatus(map[string]any{
		"branch_id":                 "branch-2",
		"poll_interval_seconds":     1,
		"max_poll_interval_seconds": 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status := stringsLower(resp["status"]); status != "succeed" {
		t.Fatalf("expected succeed status, got %s", status)
	}
	if elapsed := clock.Elapsed(); elapsed < 2*time.Second || elapsed >= handler.branchStatusTimeout {
		t.Fatalf("expected elapsed between 2s and %s, got %s", handler.branchStatusTimeout, elapsed)
	}
}

type branchReadInput struct {
	branchID string
	path     string
}

type branchReadResult struct {
	data map[string]any
	err  error
}

type fakeMCPClient struct {
	parallelExploreCalls int
	readResults          []branchReadResult
	branchReadInputs     []branchReadInput
	branchOutputInputs   []branchOutputInput
	branchOutputResult   map[string]any
	branchOutputErr      error
	getBranchFunc        func(string) (map[string]any, error)
}

type branchOutputInput struct {
	branchID   string
	fullOutput bool
}

func (f *fakeMCPClient) ParallelExplore(projectName, parentBranchID string, prompts []string, agent string, numBranches int) (map[string]any, error) {
	f.parallelExploreCalls++
	branchID := fmt.Sprintf("branch-%d", f.parallelExploreCalls)
	return map[string]any{
		"branch_id": branchID,
	}, nil
}

func (f *fakeMCPClient) GetBranch(branchID string) (map[string]any, error) {
	if f.getBranchFunc != nil {
		return f.getBranchFunc(branchID)
	}
	return map[string]any{
		"id":     branchID,
		"status": "succeed",
	}, nil
}

func (f *fakeMCPClient) BranchReadFile(branchID, filePath string) (map[string]any, error) {
	f.branchReadInputs = append(f.branchReadInputs, branchReadInput{branchID: branchID, path: filePath})
	if len(f.readResults) == 0 {
		return nil, fmt.Errorf("no stub result for branch %s", branchID)
	}
	next := f.readResults[0]
	f.readResults = f.readResults[1:]
	if next.err != nil {
		return nil, next.err
	}
	if next.data != nil {
		if errVal, ok := next.data["error"]; ok && errVal != nil {
			switch v := errVal.(type) {
			case string:
				return nil, fmt.Errorf("%s", strings.TrimSpace(v))
			case map[string]any:
				if msg, ok := v["message"].(string); ok && strings.TrimSpace(msg) != "" {
					return nil, fmt.Errorf("%s", strings.TrimSpace(msg))
				}
				return nil, fmt.Errorf("%v", v)
			default:
				return nil, fmt.Errorf("%v", v)
			}
		}
	}
	return next.data, nil
}

func (f *fakeMCPClient) BranchOutput(branchID string, fullOutput bool) (map[string]any, error) {
	f.branchOutputInputs = append(f.branchOutputInputs, branchOutputInput{branchID: branchID, fullOutput: fullOutput})
	if f.branchOutputErr != nil {
		return nil, f.branchOutputErr
	}
	if f.branchOutputResult == nil {
		return map[string]any{"output": "ok"}, nil
	}
	return f.branchOutputResult, nil
}

func notFoundErr(attempt int) error {
	return fmt.Errorf("MCP HTTP 404: attempt %d not found", attempt)
}

type fakeClock struct {
	now   time.Time
	start time.Time
}

func newFakeClock() *fakeClock {
	start := time.Unix(0, 0)
	return &fakeClock{now: start, start: start}
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

func (f *fakeClock) Sleep(d time.Duration) {
	if d < 0 {
		d = 0
	}
	f.now = f.now.Add(d)
}

func (f *fakeClock) Elapsed() time.Duration {
	return f.now.Sub(f.start)
}
