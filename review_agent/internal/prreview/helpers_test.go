package prreview

import (
	"strings"
	"testing"
)

func TestBuildPreparationPrompt(t *testing.T) {
	prompt := buildPreparationPrompt("https://github.com/org/repo/pull/42")
	for _, needle := range []string{"gh review", "Check out", "reply ONLY JSON", "https://github.com/org/repo/pull/42"} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("prompt missing %q: %q", needle, prompt)
		}
	}
}

func TestBuildReviewPromptIncludesBranch(t *testing.T) {
	prompt := buildReviewPrompt("https://github.com/org/repo/pull/42", "feature/amazing")
	if !strings.Contains(prompt, "https://github.com/org/repo/pull/42") {
		t.Fatalf("prompt missing task context: %q", prompt)
	}
	if !strings.Contains(prompt, "feature/amazing") {
		t.Fatalf("prompt missing branch mention: %q", prompt)
	}
	for _, banned := range []string{"Run 2 of 3", "Focus area", "Instructions:"} {
		if strings.Contains(prompt, banned) {
			t.Fatalf("prompt should not include %q: %q", banned, prompt)
		}
	}
}

func TestParsePreparationSummary(t *testing.T) {
	raw := "prep complete\n```json\n{\"branch\": \"feature/foo\", \"notes\": \"synced master\"}\n```"
	summary, err := parsePreparationSummary(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Branch != "feature/foo" {
		t.Fatalf("unexpected branch: %+v", summary)
	}
}

func TestParseIssueBlocks(t *testing.T) {
	raw := "```json\n[\n  {\"name\":\"ISSUE 1\",\"statement\":\"Missing null check\",\"source_branches\":[\"branch-a\",\" \",\"branch-b\"]},\n  {\"name\":\"\",\"statement\":\"\"},\n  {\"name\":\"ISSUE 2\",\"statement\":\"Race condition\"}\n]\n```"
	issues, err := parseIssueBlocks(raw)
	if err != nil {
		t.Fatalf("parseIssueBlocks error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Name != "ISSUE 1" || issues[0].Statement != "Missing null check" {
		t.Fatalf("unexpected first issue: %+v", issues[0])
	}
	if len(issues[0].Branches) != 2 || issues[0].Branches[0] != "branch-a" || issues[0].Branches[1] != "branch-b" {
		t.Fatalf("unexpected branches: %+v", issues[0].Branches)
	}
	if issues[1].Name != "ISSUE 2" {
		t.Fatalf("unexpected second issue: %+v", issues[1])
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
