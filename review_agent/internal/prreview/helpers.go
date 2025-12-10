package prreview

import (
	"errors"
	"fmt"
	"strings"
)

func buildReviewPrompt(task string) string {
	var sb strings.Builder
	sb.WriteString("You are running review_code on a GitHub PR.\n\n")
	sb.WriteString("PR to review:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nGoals:\n")
	sb.WriteString("- Identify the single most critical P0/P1 defect in this PR.\n")
	sb.WriteString("- Provide reproduction steps and a minimal failing test idea.\n")
	sb.WriteString("- If your analysis depends on system behavior (e.g., async timing), state your assumptions explicitly.\n")
	return sb.String()
}

func buildReviewerPrompt(task string, issueText string) string {
	var sb strings.Builder
	sb.WriteString("Simulate a group of senior programmers reviewing this code change.\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	sb.WriteString("Reviewer responsibilities:\n")
	sb.WriteString("- Perform logic analysis without running code.\n")
	sb.WriteString("- Trace architectural intent (Chesterton's Fence) before proposing changes.\n")
	sb.WriteString("- Stress the design with edge cases, error handling, and concurrency questions.\n")
	sb.WriteString("- Identify how the change impacts adjacent systems.\n\n")
	sb.WriteString("Evidence standards:\n")
	sb.WriteString("✓ Concrete code reasoning and design references\n")
	sb.WriteString("✗ Fabricated mocks, guesses, or \"should\" statements\n\n")
	sb.WriteString("Response format:\n")
	sb.WriteString("# VERDICT: CONFIRMED | REJECTED\n")
	sb.WriteString("Explain the reasoning, cite specific files/lines, and describe the architectural trade-offs you considered.\n")
	return sb.String()
}

func buildTesterPrompt(task string, issueText string) string {
	var sb strings.Builder
	sb.WriteString("Simulate a QA engineer who reproduces issues by running code in Pantheon.\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue to validate:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\nTester responsibilities:\n")
	sb.WriteString("- Reproduce the report with a real command or test.\n")
	sb.WriteString("- Capture exact error messages, stack traces, and failing inputs.\n")
	sb.WriteString("- Write or adapt a minimal failing test when possible.\n")
	sb.WriteString("- Only cite evidence that comes from the actual system run; fabricated mocks are invalid.\n\n")
	sb.WriteString("Document the real test failure you observed with the full command output.\n\n")
	sb.WriteString("Always run code in Pantheon before writing your verdict so the evidence is real.\n\n")
	sb.WriteString("Response format:\n")
	sb.WriteString("# VERDICT: CONFIRMED | REJECTED\n")
	sb.WriteString("Provide: steps to reproduce, command/test output, and why the result proves or disproves the issue.\n")
	return sb.String()
}

func buildExchangePrompt(role, issueText string, peerOpinion string) string {
	var sb strings.Builder
	sb.WriteString("Round 2 consensus exchange for role: ")
	sb.WriteString(role)
	sb.WriteString("\n\nRe-read the reported issue:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\nPeer transcript:\n<<<PEER>>>\n")
	sb.WriteString(peerOpinion)
	sb.WriteString("\n<<<END PEER>>>\n\n")
	sb.WriteString("Your job:\n")
	sb.WriteString("- Consider the peer's reasoning and evidence.\n")
	sb.WriteString("- Either strengthen your confirmation with new evidence or explain precisely why you still disagree.\n")
	sb.WriteString("- Make a clear call: should we post the bug (only if both roles CONFIRMED) or hold back?\n\n")
	sb.WriteString("Response format:\n")
	sb.WriteString("# VERDICT: CONFIRMED | REJECTED\n")
	sb.WriteString("Explain whether the peer's findings changed your view and cite any new evidence gathered during this exchange.\n")
	return sb.String()
}

func extractVerdict(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", errors.New("empty transcript")
	}
	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		if clean == "" {
			continue
		}
		upper := strings.ToUpper(clean)
		idx := strings.Index(upper, "VERDICT:")
		if idx == -1 {
			continue
		}
		value := strings.TrimSpace(upper[idx+len("VERDICT:"):])
		switch value {
		case verdictConfirmed:
			return verdictConfirmed, nil
		case verdictRejected:
			return verdictRejected, nil
		case "":
			continue
		default:
			return "", fmt.Errorf("unsupported verdict %q", value)
		}
	}
	return "", errors.New("verdict directive not found")
}

func extractJSONBlock(raw string) string {
	trimmed := strings.TrimSpace(raw)
	startObj := strings.Index(trimmed, "{")
	startArr := strings.Index(trimmed, "[")
	start := -1
	end := -1
	if startArr >= 0 && (startObj == -1 || startArr < startObj) {
		start = startArr
		end = strings.LastIndex(trimmed, "]")
	} else if startObj >= 0 {
		start = startObj
		end = strings.LastIndex(trimmed, "}")
	}
	if start >= 0 && end >= start {
		return trimmed[start : end+1]
	}
	return trimmed
}
