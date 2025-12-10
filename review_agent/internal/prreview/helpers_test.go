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

func TestBuildReviewerPromptIncludesLogicReviewDirectives(t *testing.T) {
	prompt := buildReviewerPrompt("task-123", "suspect issue")
	for _, phrase := range []string{
		"Simulate a group of senior programmers",
		"Analyze the code logic for correctness",
		"edge cases and error handling",
		"Chesterton's Fence",
		"Identify potential design issues",
		"CONFIRMED | REJECTED",
		"task-123",
		"suspect issue",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("reviewer prompt missing %q: %s", phrase, prompt)
		}
	}
}

func TestBuildTesterPromptFocusesOnReproduction(t *testing.T) {
	prompt := buildTesterPrompt("task-abc", "bug text here")
	for _, phrase := range []string{
		"Simulate a QA engineer who reproduces issues by running code",
		"Attempt to reproduce the reported issue",
		"Write and run a minimal failing test",
		"Trace actual execution paths",
		"Collect real error messages",
		"CONFIRMED (with test evidence)",
		"task-abc",
		"bug text here",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("tester prompt missing %q: %s", phrase, prompt)
		}
	}
}

func TestBuildExchangePromptSharesPeerOpinion(t *testing.T) {
	prompt := buildExchangePrompt("tester", "issue body", "peer says hi")
	for _, phrase := range []string{
		"Round 2",
		"tester",
		"peer says hi",
		"<<<PEER_OPINION>>>",
		"CONFIRMED | REJECTED",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("exchange prompt missing %q: %s", phrase, prompt)
		}
	}
}

func TestParseVerdictFromTranscript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "plain confirmed",
			text: `
			# VERDICT: CONFIRMED
			Rest of the body
			`,
			want: verdictConfirmed,
		},
		{
			name: "plain rejected",
			text: "# VERDICT: REJECTED\nreasoning",
			want: verdictRejected,
		},
		{
			name: "confirmed with tester qualifier",
			text: "# VERDICT: CONFIRMED (with test evidence)\ncommands",
			want: verdictConfirmed,
		},
		{
			name: "confirmed qualifier may include pipe character",
			text: "# VERDICT: CONFIRMED (cat foo | grep bar)",
			want: verdictConfirmed,
		},
		{
			name: "rejected with qualifier",
			text: "# VERDICT: REJECTED (not enough evidence)",
			want: verdictRejected,
		},
		{
			name: "confirmed with colon qualifier",
			text: "# VERDICT: CONFIRMED: reproduction attached",
			want: verdictConfirmed,
		},
		{
			name: "confirmed with hyphen qualifier",
			text: "# VERDICT: CONFIRMED- log snippet follows",
			want: verdictConfirmed,
		},
		{
			name: "skips prompt instructions in favor of actual verdict",
			text: `
			PROMPT REMINDER
			# VERDICT: CONFIRMED | REJECTED
			# VERDICT: REJECTED (not enough evidence)
			`,
			want: verdictRejected,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			verdict, err := parseVerdictFromTranscript(tt.text)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if verdict != tt.want {
				t.Fatalf("expected verdict %q, got %q", tt.want, verdict)
			}
		})
	}
}

func TestParseVerdictFromTranscriptErrorsWithoutMarker(t *testing.T) {
	if _, err := parseVerdictFromTranscript("no verdict here"); err == nil {
		t.Fatalf("expected error when verdict marker is missing")
	}
}

func TestParseVerdictFromTranscriptRejectsUnknownVerdict(t *testing.T) {
	if _, err := parseVerdictFromTranscript("# VERDICT: MAYBE (needs more info)"); err == nil {
		t.Fatalf("expected error for unknown verdict token")
	}
}

func TestParseVerdictFromTranscriptRejectsPromptInstructions(t *testing.T) {
	if _, err := parseVerdictFromTranscript("# VERDICT: CONFIRMED | REJECTED"); err == nil {
		t.Fatalf("expected error when only prompt instructions are present")
	}
}

func TestParseVerdictFromTranscriptErrorsOnEmptyVerdictLine(t *testing.T) {
	if _, err := parseVerdictFromTranscript("# VERDICT:   "); err == nil {
		t.Fatalf("expected error when verdict line is empty")
	}
}

func TestParseVerdictFromTranscriptRejectsUnsupportedQualifier(t *testing.T) {
	if _, err := parseVerdictFromTranscript("# VERDICT: CONFIRMED / additional context"); err == nil {
		t.Fatalf("expected error for unsupported qualifier prefix")
	}
}

func TestEvaluateRoundOneOutcome(t *testing.T) {
	reviewer := Transcript{Verdict: verdictRejected}
	tester := Transcript{Verdict: verdictRejected}
	outcome := evaluateRoundOne(reviewer, tester, false)
	if outcome.NeedExchange {
		t.Fatalf("should not require exchange when both reject")
	}
	if outcome.Status != commentUnresolved {
		t.Fatalf("expected unresolved status when both reject, got %q", outcome.Status)
	}

	reviewer.Verdict = verdictConfirmed
	tester.Verdict = verdictConfirmed
	outcome = evaluateRoundOne(reviewer, tester, true)
	if outcome.NeedExchange {
		t.Fatalf("unexpected exchange when both confirm with consensus true")
	}
	if outcome.Status != commentConfirmed {
		t.Fatalf("expected confirmed status when consensus true, got %q", outcome.Status)
	}

	outcome = evaluateRoundOne(reviewer, tester, false)
	if !outcome.NeedExchange || outcome.Status != "" {
		t.Fatalf("expected exchange + no status when consensus false, got %+v", outcome)
	}

	tester.Verdict = verdictRejected
	outcome = evaluateRoundOne(reviewer, tester, false)
	if !outcome.NeedExchange {
		t.Fatalf("expected exchange when verdicts differ")
	}
	if outcome.Status != "" {
		t.Fatalf("status should be empty when further processing required")
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
