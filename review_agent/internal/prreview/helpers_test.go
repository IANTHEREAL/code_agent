package prreview

import (
	"strings"
	"testing"
)

func TestParseIssueBlocks(t *testing.T) {
	raw := `
ISSUE 1: Missing null check in foo
Context line

ISSUE 2 - Race condition
Details here

ISSUE 3: 
`
	issues := parseIssueBlocks(raw)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Name != "ISSUE 1" {
		t.Fatalf("unexpected name %q", issues[0].Name)
	}
	want := "Missing null check in foo\nContext line"
	if issues[0].Statement != want {
		t.Fatalf("unexpected statement %q", issues[0].Statement)
	}
	if issues[1].Name != "ISSUE 2" {
		t.Fatalf("unexpected name %q", issues[1].Name)
	}
	if !strings.Contains(issues[1].Statement, "Race condition") {
		t.Fatalf("second issue missing content: %q", issues[1].Statement)
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
