package prreview

import (
	"strings"
	"testing"
)

func TestBuildIssueFinderPromptContainsInstructions(t *testing.T) {
	task := "https://github.com/org/repo/pull/42"
	got := buildIssueFinderPrompt(task)

	required := []string{
		"Task: " + task,
		"Review the code changes against the base branch",
		"git merge-base HEAD BASE_BRANCH",
		"git diff MERGE_BASE_SHA",
		"Provide prioritized, actionable findings",
	}

	for _, req := range required {
		if !strings.Contains(got, req) {
			t.Errorf("prompt missing required text: %q", req)
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

func TestBuildAlignmentPromptContainsInputs(t *testing.T) {
	alpha := Transcript{Text: "A says # VERDICT: CONFIRMED"}
	beta := Transcript{Text: "B says # VERDICT: CONFIRMED"}
	issue := "ISSUE: sample"
	prompt := buildAlignmentPrompt(issue, alpha, beta)
	required := []string{
		issue,
		alpha.Text,
		beta.Text,
		"Reply ONLY JSON",
		"agree=true",
	}
	for _, needle := range required {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("alignment prompt missing %q: %q", needle, prompt)
		}
	}
}

func TestExtractTranscriptVerdict(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		want     string
		wantFind bool
	}{
		{
			name:     "confirmed marker",
			input:    "# VERDICT: CONFIRMED\n\n## Reasoning\nok",
			want:     "confirmed",
			wantFind: true,
		},
		{
			name:     "rejected marker lower-case",
			input:    "   # verdict: rejected\nDetails...",
			want:     "rejected",
			wantFind: true,
		},
		{
			name:     "bracketed marker",
			input:    "# VERDICT: [CONFIRMED]\nEvidence",
			want:     "confirmed",
			wantFind: true,
		},
		{
			name:     "ignores quoted marker",
			input:    "> # VERDICT: CONFIRMED\n\nNo explicit marker here",
			want:     "",
			wantFind: false,
		},
		{
			name:     "no marker",
			input:    "I think this is a bug but forgot the header",
			want:     "",
			wantFind: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, ok := extractTranscriptVerdict(tc.input)
			if ok != tc.wantFind {
				t.Fatalf("expected found=%v, got %v (decision=%+v)", tc.wantFind, ok, decision)
			}
			if !ok {
				return
			}
			if decision.Verdict != tc.want {
				t.Fatalf("expected verdict %q, got %q", tc.want, decision.Verdict)
			}
			if decision.Reason == "" {
				t.Fatalf("expected non-empty reason")
			}
		})
	}
}
