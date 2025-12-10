package prreview

import (
	"strings"
	"testing"
)

func TestEvaluatePanelConsensusRequiresUnanimousConfirmation(t *testing.T) {
	reviewer := Transcript{Agent: "reviewer", Verdict: verdictConfirmed}
	tester := Transcript{Agent: "tester", Verdict: verdictRejected}

	result := evaluatePanelConsensus(reviewer, tester)

	if result.Agree {
		t.Fatalf("expected disagreement result, got %+v", result)
	}
	if result.Status != commentUnresolved {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.Explanation == "" || result.Verdict != "" {
		t.Fatalf("expected explanation text and empty verdict, got %+v", result)
	}
}

func TestEvaluatePanelConsensusStopsWhenBothReject(t *testing.T) {
	reviewer := Transcript{Agent: "reviewer", Verdict: verdictRejected}
	tester := Transcript{Agent: "tester", Verdict: verdictRejected}

	result := evaluatePanelConsensus(reviewer, tester)

	if !result.Agree {
		t.Fatalf("expected agreement on rejection, got %+v", result)
	}
	if result.Status != commentUnresolved {
		t.Fatalf("double rejection should map to no-post unresolved status, got %s", result.Status)
	}
	if !strings.Contains(result.Explanation, "REJECTED") {
		t.Fatalf("explanation should mention rejection: %s", result.Explanation)
	}
}

func TestEvaluatePanelConsensusConfirmsWhenBothConfirm(t *testing.T) {
	reviewer := Transcript{Agent: "reviewer", Verdict: verdictConfirmed}
	tester := Transcript{Agent: "tester", Verdict: verdictConfirmed}

	result := evaluatePanelConsensus(reviewer, tester)

	if !result.Agree {
		t.Fatalf("expected unanimous agreement, got %+v", result)
	}
	if result.Status != commentConfirmed || result.Verdict != verdictConfirmed {
		t.Fatalf("unexpected confirmation verdict %+v", result)
	}
	if !strings.Contains(result.Explanation, "unanimous") {
		t.Fatalf("confirmation explanation missing unanimity note: %s", result.Explanation)
	}
}

func TestRunExchangeUsesPeerBranchID(t *testing.T) {
	runner := &Runner{opts: Options{ParentBranchID: "parent"}}
	var capturedParent string
	var capturedRound int
	runner.runRoleFn = func(role, prompt string, round int, parent string) (Transcript, error) {
		capturedParent = parent
		capturedRound = round
		return Transcript{Agent: role, Round: round, BranchID: "child-branch", Text: "response"}, nil
	}

	peer := Transcript{Agent: roleTester, BranchID: "peer-branch", Text: "peer text", Round: 1}
	current := Transcript{Agent: roleReviewer, BranchID: "current-branch", Round: 1}

	result, err := runner.runExchange(roleReviewer, "issue text", peer, current)
	if err != nil {
		t.Fatalf("runExchange returned error: %v", err)
	}
	if capturedParent != peer.BranchID {
		t.Fatalf("expected exchange to branch from peer %q, got %q", peer.BranchID, capturedParent)
	}
	if capturedRound != current.Round+1 {
		t.Fatalf("expected next round %d, got %d", current.Round+1, capturedRound)
	}
	if result.Round != capturedRound {
		t.Fatalf("transcript round should match runRole result: %d vs %d", result.Round, capturedRound)
	}
}

func TestRunExchangeFallsBackToCurrentBranchWhenPeerMissing(t *testing.T) {
	runner := &Runner{opts: Options{ParentBranchID: "parent"}}
	var capturedParent string
	runner.runRoleFn = func(role, prompt string, round int, parent string) (Transcript, error) {
		capturedParent = parent
		return Transcript{Agent: role, Round: round, BranchID: "child-branch"}, nil
	}

	peer := Transcript{Agent: roleTester, Text: "peer text", Round: 2}
	current := Transcript{Agent: roleReviewer, BranchID: "current-branch", Round: 2}

	_, err := runner.runExchange(roleReviewer, "issue text", peer, current)
	if err != nil {
		t.Fatalf("runExchange returned error: %v", err)
	}
	if capturedParent != current.BranchID {
		t.Fatalf("expected fallback to current branch %q, got %q", current.BranchID, capturedParent)
	}
}
