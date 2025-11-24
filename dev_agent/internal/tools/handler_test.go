package tools

import (
	"errors"
	"reflect"
	"testing"
)

type fakeAgentClient struct {
	parallelExploreResp map[string]any
	parallelExploreErr  error

	getBranchResp map[string]any
	getBranchErr  error

	branchOutputResp map[string]any
	branchOutputErr  error
}

func (f *fakeAgentClient) ParallelExplore(projectName, parentBranchID string, prompts []string, agent string, numBranches int) (map[string]any, error) {
	return f.parallelExploreResp, f.parallelExploreErr
}

func (f *fakeAgentClient) GetBranch(branchID string) (map[string]any, error) {
	return f.getBranchResp, f.getBranchErr
}

func (f *fakeAgentClient) BranchReadFile(branchID, filePath string) (map[string]any, error) {
	return nil, errors.New("unexpected call")
}

func (f *fakeAgentClient) BranchOutput(branchID string) (map[string]any, error) {
	return f.branchOutputResp, f.branchOutputErr
}

func TestExecuteAgentReturnsBranchOutputResponse(t *testing.T) {
	fake := &fakeAgentClient{
		parallelExploreResp: map[string]any{"branch_id": "branch-123"},
		getBranchResp:       map[string]any{"branch_id": "branch-123", "status": "succeed", "output": "status output"},
		branchOutputResp:    map[string]any{"output": "branch output", "extra": "details"},
	}

	handler := NewToolHandler(fake, "proj", "start-branch")
	args := map[string]any{
		"agent":            "codex",
		"prompt":           "do work",
		"parent_branch_id": "start-branch",
	}

	result, err := handler.executeAgent(args)
	if err != nil {
		t.Fatalf("executeAgent failed: %v", err)
	}

	if got := result["response"].(string); got != "branch output" {
		t.Fatalf("expected response to come from branch output, got %q", got)
	}

	bo, ok := result["branch_output"].(map[string]any)
	if !ok {
		t.Fatalf("expected branch_output map in result")
	}
	if !reflect.DeepEqual(bo, fake.branchOutputResp) {
		t.Fatalf("branch_output mismatch: got %+v", bo)
	}
}

func TestExecuteAgentFallsBackWhenBranchOutputFails(t *testing.T) {
	fake := &fakeAgentClient{
		parallelExploreResp: map[string]any{"branch_id": "branch-123"},
		getBranchResp:       map[string]any{"branch_id": "branch-123", "status": "succeed", "output": "status output"},
		branchOutputErr:     errors.New("rpc failure"),
	}

	handler := NewToolHandler(fake, "proj", "start-branch")
	args := map[string]any{
		"agent":            "codex",
		"prompt":           "do work",
		"parent_branch_id": "start-branch",
	}

	result, err := handler.executeAgent(args)
	if err != nil {
		t.Fatalf("executeAgent failed: %v", err)
	}

	if got := result["response"].(string); got != "status output" {
		t.Fatalf("expected fallback response, got %q", got)
	}

	if _, ok := result["branch_output"].(map[string]any); ok {
		t.Fatalf("did not expect branch_output map when call fails")
	}

	errMsg, _ := result["branch_output_error"].(string)
	if errMsg == "" || errMsg != "rpc failure" {
		t.Fatalf("expected branch_output_error to propagate failure message, got %q", errMsg)
	}
}
