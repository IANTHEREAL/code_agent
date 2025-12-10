package prreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func buildIssueFinderPrompt(task string) string {
	// NOTE: The Codex review flow expects this sentinel string to trigger its built-in
	// prompt template; see https://github.com/openai/codex/issues/6432 for context.
	return "base-branch main"
}

// buildReviewerPrompt creates the prompt for the Reviewer role (logic analysis).
func buildLogicAnalystPrompt(task string, issueText string) string {
	var sb strings.Builder
	sb.WriteString("Verification Role: REVIEWER\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	sb.WriteString("YOUR ROLE: Analyze code logic to determine if this is a valid bug.\n\n")
	sb.WriteString("Simulate a group of senior programmers reviewing this code change.\n\n")
	sb.WriteString("Their task:\n")
	sb.WriteString("- Analyze the code logic for correctness\n")
	sb.WriteString("- Check for edge cases and error handling\n")
	sb.WriteString("- Understand the architectural intent (Chesterton's Fence: why does this code exist?)\n")
	sb.WriteString("- Identify if the reported issue is a real bug or a misunderstanding\n\n")
	sb.WriteString("EVIDENCE STANDARDS:\n")
	sb.WriteString("✓ Valid: Code traces, execution paths, architectural analysis\n")
	sb.WriteString("✗ Invalid: Assumptions, intuitions, \"this looks wrong\"\n\n")
	sb.WriteString("RESPONSE FORMAT:\n")
	sb.WriteString("Start with: # VERDICT: [CONFIRMED | REJECTED]\n\n")
	sb.WriteString("Then provide:\n")
	sb.WriteString("## Reasoning\n")
	sb.WriteString("<Your analysis of the code logic>\n\n")
	sb.WriteString("## Evidence\n")
	sb.WriteString("<Code traces or architectural analysis supporting your verdict>\n")
	return sb.String()
}

// buildTesterPrompt creates the prompt for the Tester role (reproduction).
func buildTesterPrompt(task string, issueText string) string {
	var sb strings.Builder
	sb.WriteString("Verification Role: TESTER\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	sb.WriteString("YOUR ROLE: Reproduce the issue by actually running code.\n\n")
	sb.WriteString("Simulate a QA engineer who verifies bugs by running real tests.\n\n")
	sb.WriteString("Their task:\n")
	sb.WriteString("- Attempt to reproduce the reported issue\n")
	sb.WriteString("- Write and run a minimal failing test\n")
	sb.WriteString("- Trace actual execution paths\n")
	sb.WriteString("- Collect real error messages (not assumptions)\n\n")
	sb.WriteString("CRITICAL: You MUST actually run code to collect evidence.\n")
	sb.WriteString("Do NOT fabricate test results or mock behavior.\n\n")
	sb.WriteString("EVIDENCE STANDARDS:\n")
	sb.WriteString("✓ Valid: Actual test output, real error messages, execution traces\n")
	sb.WriteString("✗ Invalid: Self-created mocks, assumed behavior, \"should\" statements\n\n")
	sb.WriteString("RESPONSE FORMAT:\n")
	sb.WriteString("Start with: # VERDICT: [CONFIRMED | REJECTED]\n\n")
	sb.WriteString("Then provide:\n")
	sb.WriteString("## Reproduction Steps\n")
	sb.WriteString("<What you did to reproduce>\n\n")
	sb.WriteString("## Test Evidence\n")
	sb.WriteString("<Actual test output or error messages>\n")
	return sb.String()
}

// buildExchangePrompt creates the prompt for Round 2 (exchange opinions).
func buildExchangePrompt(role string, task string, issueText string, selfOpinion string, peerOpinion string) string {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	displayRole := strings.ToUpper(role)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Verification Role: %s (Round 2 - Exchange)\n\n", displayRole))
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	sb.WriteString("YOUR PREVIOUS OPINION:\n<<<SELF>>>\n")
	sb.WriteString(selfOpinion)
	sb.WriteString("\n<<<END SELF>>>\n\n")
	sb.WriteString("PEER'S OPINION:\n<<<PEER>>>\n")
	sb.WriteString(peerOpinion)
	sb.WriteString("\n<<<END PEER>>>\n\n")
	sb.WriteString("ROLE REMINDER:\n")
	switch normalizedRole {
	case "reviewer":
		sb.WriteString("- You remain the logic analysis reviewer. Focus on code logic and architecture.\n")
		sb.WriteString("- Do NOT claim you ran tests; rely on reasoning and Chesterton's Fence thinking.\n")
	case "tester":
		sb.WriteString("- You remain the tester. You must run code and capture actual execution output.\n")
		sb.WriteString("- Provide real execution evidence such as logs or failing test output.\n")
	default:
		sb.WriteString("- Stay consistent with your original role responsibilities.\n")
	}
	sb.WriteString("\n")
	sb.WriteString("YOUR TASK:\n")
	sb.WriteString("You previously reviewed this issue. Now you have seen your peer's analysis.\n")
	sb.WriteString("- Consider their evidence and reasoning\n")
	sb.WriteString("- Re-evaluate your position\n")
	sb.WriteString("- You may change your verdict if their evidence is convincing\n")
	sb.WriteString("- You may maintain your verdict if you find flaws in their reasoning\n\n")
	sb.WriteString("RESPONSE FORMAT:\n")
	sb.WriteString("Start with: # VERDICT: [CONFIRMED | REJECTED]\n\n")
	sb.WriteString("Then provide:\n")
	sb.WriteString("## Response to Peer\n")
	sb.WriteString("<Address their key points>\n\n")
	sb.WriteString("## Final Reasoning\n")
	sb.WriteString("<Your updated analysis>\n")
	return sb.String()
}

type verdictDecision struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

func buildVerdictPrompt(transcript string) string {
	var sb strings.Builder
	sb.WriteString("You are analyzing an agent transcript from a PR verification workflow.\n")
	sb.WriteString("Determine whether the agent ultimately CONFIRMED the reported issue or REJECTED it.\n\n")
	sb.WriteString("Transcript:\n<<<TRANSCRIPT>>>\n")
	sb.WriteString(transcript)
	sb.WriteString("\n<<<END TRANSCRIPT>>>\n\n")
	sb.WriteString("Reply ONLY JSON: {\"verdict\":\"confirmed|rejected\",\"reason\":\"<original explanation>\"}.\n")
	sb.WriteString("Use \"confirmed\" only if the agent clearly states the issue is real and provides evidence.\n")
	sb.WriteString("Otherwise return \"rejected\".\n")
	return sb.String()
}

func parseVerdictDecision(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("empty verdict response")
	}
	jsonBlock := extractJSONBlock(trimmed)
	var decision verdictDecision
	if err := json.Unmarshal([]byte(jsonBlock), &decision); err != nil {
		return "", err
	}
	verdict := strings.ToLower(strings.TrimSpace(decision.Verdict))
	switch verdict {
	case "confirmed", "rejected":
		return verdict, nil
	default:
		return "", fmt.Errorf("invalid verdict %q", decision.Verdict)
	}
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
