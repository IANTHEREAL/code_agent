package tools

import (
	"errors"
	"fmt"
	"testing"
)

func TestExecuteAgentReviewCodeRetriesMissingLog(t *testing.T) {
	client := &fakeMCPClient{
		readResults: []branchReadResult{
			{err: notFoundErr(1)},
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
}

func (f *fakeMCPClient) ParallelExplore(projectName, parentBranchID string, prompts []string, agent string, numBranches int) (map[string]any, error) {
	f.parallelExploreCalls++
	branchID := fmt.Sprintf("branch-%d", f.parallelExploreCalls)
	return map[string]any{
		"branch_id": branchID,
	}, nil
}

func (f *fakeMCPClient) GetBranch(branchID string) (map[string]any, error) {
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
	return next.data, next.err
}

func notFoundErr(attempt int) error {
	return fmt.Errorf("MCP HTTP 404: attempt %d not found", attempt)
}
