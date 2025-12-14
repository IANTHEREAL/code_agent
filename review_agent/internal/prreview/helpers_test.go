package prreview

import (
	"strings"
	"testing"
)

func TestBuildReviewPromptFocusesSingleCriticalIssue(t *testing.T) {
	prompt := buildReviewPrompt("https://github.com/org/repo/pull/42")
	for _, needle := range []string{
		"https://github.com/org/repo/pull/42",
		"single most critical",
		"P0/P1",
	} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("prompt missing %q: %q", needle, prompt)
		}
	}
}

func TestBuildReviewerPromptMentionsLogicPanel(t *testing.T) {
	prompt := buildReviewerPrompt("task context", "issue text")
	expect := []string{
		"Simulate a group of senior programmers",
		"logic analysis",
		"edge cases",
		"issue text",
		"# VERDICT",
	}
	for _, needle := range expect {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("reviewer prompt missing %q: %s", needle, prompt)
		}
	}
}

func TestBuildTesterPromptFocusesOnReproduction(t *testing.T) {
	prompt := buildTesterPrompt("task context", "issue text")
	expect := []string{
		"Simulate a QA engineer",
		"run code in Pantheon",
		"real test failure",
		"issue text",
		"# VERDICT",
	}
	for _, needle := range expect {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("tester prompt missing %q: %s", needle, prompt)
		}
	}
}

func TestBuildExchangePromptIncludesPeerOpinion(t *testing.T) {
	peer := "Peer says CONFIRMED"
	prompt := buildExchangePrompt("reviewer", "issue text", peer)
	if !strings.Contains(prompt, "Round 2") {
		t.Fatalf("exchange prompt missing round 2 context: %s", prompt)
	}
	if !strings.Contains(prompt, "<<<PEER>>>") || !strings.Contains(prompt, peer) {
		t.Fatalf("exchange prompt missing peer transcript: %s", prompt)
	}
}

func TestExtractVerdictParsesDirective(t *testing.T) {
	text := `
Random context
# VERDICT: confirmed
Details afterwards`
	verdict, err := extractVerdict(text)
	if err != nil {
		t.Fatalf("extractVerdict returned error: %v", err)
	}
	if verdict != "CONFIRMED" {
		t.Fatalf("expected CONFIRMED verdict, got %s", verdict)
	}
}

func TestExtractVerdictFailsWithoutDirective(t *testing.T) {
	_, err := extractVerdict("no verdict here")
	if err == nil {
		t.Fatal("expected error when verdict directive absent")
	}
}
