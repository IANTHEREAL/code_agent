package prreview

import (
	"fmt"
	"testing"

	b "review_agent/internal/brain"
	tools "review_agent/internal/tools"
)

type fakeAgentClient struct {
	outputs []string
	next    int
	byID    map[string]string
}

func newFakeAgentClient(outputs []string) *fakeAgentClient {
	return &fakeAgentClient{
		outputs: outputs,
		byID:    map[string]string{},
	}
}

func (c *fakeAgentClient) ParallelExplore(projectName, parentBranchID string, prompts []string, agent string, numBranches int) (map[string]any, error) {
	c.next++
	branchID := fmt.Sprintf("branch_%d", c.next)
	out := ""
	if c.next-1 < len(c.outputs) {
		out = c.outputs[c.next-1]
	}
	c.byID[branchID] = out
	return map[string]any{
		"branch_id": branchID,
	}, nil
}

func (c *fakeAgentClient) GetBranch(branchID string) (map[string]any, error) {
	return map[string]any{
		"id":            branchID,
		"status":        "succeed",
		"latest_snap_id": fmt.Sprintf("%s_snap", branchID),
	}, nil
}

func (c *fakeAgentClient) BranchReadFile(branchID string, filePath string) (map[string]any, error) {
	return map[string]any{}, fmt.Errorf("not implemented")
}

func (c *fakeAgentClient) BranchOutput(branchID string, fullOutput bool) (map[string]any, error) {
	return map[string]any{
		"output": c.byID[branchID],
	}, nil
}

func TestConfirmIssueDoesNotConfirmWhenTranscriptsMisaligned(t *testing.T) {
	reviewer := "# VERDICT: CONFIRMED\n\nClaim: issueText describes defect A\nAnchor: alpha.go:10\n\n## Reasoning\nConfirmed Defect A."
	tester := "# VERDICT: CONFIRMED\n\nClaim: issueText describes defect B\nAnchor: beta.go:20\n\n## Reproduction Steps\nConfirmed Defect B."
	reviewerR2 := "# VERDICT: CONFIRMED\n\nClaim: defect A\nAnchor: alpha.go:10\n\n## Response to Peer\nStill A.\n\n## Final Reasoning\nStill A."
	testerR2 := "# VERDICT: CONFIRMED\n\nClaim: defect B\nAnchor: beta.go:20\n\n## Response to Peer\nStill B.\n\n## Final Reasoning\nStill B."
	client := newFakeAgentClient([]string{reviewer, tester, reviewerR2, testerR2})
	handler := tools.NewToolHandler(client, "proj", "start", "")
	runner, err := NewRunner(&b.LLMBrain{}, handler, nil, Options{
		Task:           "task",
		ProjectName:    "proj",
		ParentBranchID: "start",
		WorkspaceDir:   "",
	})
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	runner.alignmentOverride = func(issueText string, alpha Transcript, beta Transcript) (alignmentVerdict, error) {
		return alignmentVerdict{Agree: false, Explanation: "test: misaligned"}, nil
	}

	report, err := runner.confirmIssue("ISSUE: example", "start")
	if err != nil {
		t.Fatalf("confirmIssue error: %v", err)
	}
	if report.Status == commentConfirmed {
		t.Fatalf("expected unresolved when reviewer/tester confirm different anchors; got status=%q explanation=%q", report.Status, report.VerdictExplanation)
	}
}
