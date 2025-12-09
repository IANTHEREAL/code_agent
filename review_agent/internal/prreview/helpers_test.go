package prreview

import (
	"strings"
	"testing"
)

func TestBuildReviewPromptFocusesSingleCriticalIssue(t *testing.T) {
	prompt := buildIssueFinderPrompt("https://github.com/org/repo/pull/42")
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

func TestBuildReviewerPromptContainsRoleDirectives(t *testing.T) {
	prompt := buildLogicAnalystPrompt("some task", "some issue")
	requiredPhrases := []string{
		"REVIEWER",
		"Simulate a group of senior programmers",
		"Chesterton's Fence",
		"VERDICT",
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("reviewer prompt missing directive: %q", phrase)
		}
	}
}

func TestBuildTesterPromptContainsRoleDirectives(t *testing.T) {
	prompt := buildTesterPrompt("some task", "some issue")
	requiredPhrases := []string{
		"TESTER",
		"Simulate a QA engineer",
		"MUST actually run code",
		"Do NOT fabricate",
		"VERDICT",
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("tester prompt missing directive: %q", phrase)
		}
	}
}

func TestBuildExchangePromptIncludesSelfPeerAndReviewerGuidance(t *testing.T) {
	prompt := buildExchangePrompt("reviewer", "task", "issue", "my old verdict", "peer said hello")
	required := []string{
		"my old verdict",
		"peer said hello",
		"YOUR PREVIOUS OPINION",
		"PEER'S OPINION",
		"logic analysis",
		"Do NOT claim you ran tests",
	}
	for _, needle := range required {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("reviewer exchange prompt missing %q: %q", needle, prompt)
		}
	}
}

func TestBuildExchangePromptProvidesTesterGuidance(t *testing.T) {
	prompt := buildExchangePrompt("tester", "task", "issue", "my reproduction log", "peer logic view")
	required := []string{
		"my reproduction log",
		"peer logic view",
		"YOUR PREVIOUS OPINION",
		"PEER'S OPINION",
		"run code",
		"real execution evidence",
	}
	for _, needle := range required {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("tester exchange prompt missing %q: %q", needle, prompt)
		}
	}
}

func TestBuildVerdictPromptContainsTranscript(t *testing.T) {
	resp := "# VERDICT: CONFIRMED\nEvidence here"
	prompt := buildVerdictPrompt(resp)
	if !strings.Contains(prompt, resp) {
		t.Fatalf("prompt missing transcript: %q", prompt)
	}
	if !strings.Contains(prompt, "Reply ONLY JSON") {
		t.Fatalf("prompt missing JSON instruction: %q", prompt)
	}
}

func TestParseVerdictDecision(t *testing.T) {
	raw := "```json\n{\"verdict\": \"confirmed\", \"reason\": \"explicit marker\"}\n```"
	verdict, err := parseVerdictDecision(raw)
	if err != nil {
		t.Fatalf("parseVerdictDecision error: %v", err)
	}
	if verdict != "confirmed" {
		t.Fatalf("unexpected verdict: %s", verdict)
	}

	bad := "```json\n{\"verdict\": \"unknown\"}\n```"
	if _, err := parseVerdictDecision(bad); err == nil {
		t.Fatal("expected error for invalid verdict")
	}
}
