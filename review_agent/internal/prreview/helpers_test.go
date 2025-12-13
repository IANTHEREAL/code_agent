package prreview

import (
	"strings"
	"testing"
)

func TestBuildReviewPromptUsesCodexSentinel(t *testing.T) {
	// NOTE: buildIssueFinderPrompt intentionally returns the literal sentinel
	// expected by Codex review (see https://github.com/openai/codex/issues/6432).
	if got := buildIssueFinderPrompt("https://github.com/org/repo/pull/42"); got != "base-branch main" {
		t.Fatalf("expected codex sentinel, got %q", got)
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
		"include the key command or code snippet",
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
	decision, err := parseVerdictDecision(raw)
	if err != nil {
		t.Fatalf("parseVerdictDecision error: %v", err)
	}
	if decision.Verdict != "confirmed" {
		t.Fatalf("unexpected verdict: %s", decision.Verdict)
	}
	if decision.Reason != "explicit marker" {
		t.Fatalf("unexpected reason: %s", decision.Reason)
	}

	bad := "```json\n{\"verdict\": \"unknown\"}\n```"
	if _, err := parseVerdictDecision(bad); err == nil {
		t.Fatal("expected error for invalid verdict")
	}
}
