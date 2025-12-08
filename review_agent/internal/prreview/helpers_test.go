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

func TestBuildVerifierPromptContainsLinusDirectives(t *testing.T) {
	prompt := buildVerifierPrompt("verifier-alpha", "some task", "some issue", "", 1)
	requiredPhrases := []string{
		"Linus Torvalds persona",
		"Chesterton's Fence",
		"Safety First",
		"Architectural Intent",
		"Reproduction Summary",
		"core test snippet",
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("verifier prompt missing core directive: %q", phrase)
		}
	}
}

func TestParseConsensus(t *testing.T) {
	raw := "```json\n{\"agree\": true, \"explanation\": \"same failure\"}\n```"
	verdict, err := parseConsensus(raw)
	if err != nil {
		t.Fatalf("parseConsensus error: %v", err)
	}
	if !verdict.Agree || verdict.Explanation != "same failure" {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}
