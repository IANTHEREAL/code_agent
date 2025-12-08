package prreview

import (
	"encoding/json"
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

func buildVerifierPrompt(label, task string, issueText string, peerTranscript string, round int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are %s, an elite Systems Architect acting as an Issue Verifier (Linus Torvalds persona).\n\n", label))
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	sb.WriteString("GOAL: Analyze the issue to determine if it is a valid, critical bug.\n\n")
	sb.WriteString("INSTRUCTIONS:\n")
	sb.WriteString("1. Code Reasoning Bug (Chesterton's Fence): Before you call it a bug, understand WHY the code is there. Is it a feature disguised as a bug? Is it protecting against a race condition? Explain the architectural intent.\n")
	sb.WriteString("2. Safety First: If we 'fix' this, what safety guarantees do we lose? Does it introduce a race condition or a dirty read?\n")
	sb.WriteString("3. Write Tests (Proof of Work): You MUST write a minimal unit test (in the appropriate language) to the disk and run it to verify your finding. Do NOT skip this step.\n")
	sb.WriteString("4. Be Skeptical but Smart: Assume the code might be right and the issue reporter is missing the big picture. Only confirm if you can prove it is BROKEN and the fix is SAFE.\n")
	sb.WriteString("\nFirst, follow the instructions above to verify the bug. Then, output your report following these requirements:\n")
	sb.WriteString("\nREPORT REQUIREMENTS:\n")
	sb.WriteString("- If confirmed: You MUST follow this EXACT format:\n")
	sb.WriteString("  ## [P1] <Short Title>\n\n")
	sb.WriteString("  ### Issue Location\n")
	sb.WriteString("  `path/to/file:line_number`\n\n")
	sb.WriteString("  ### Problem Summary\n")
	sb.WriteString("  <Concise description of the bug>\n\n")
	sb.WriteString("  ### Root Cause Analysis\n")
	sb.WriteString("  <Detailed explanation tracing the values and logic. Use tables or step-by-step lists if helpful.>\n\n")
	sb.WriteString("  ### Impact\n")
	sb.WriteString("  <What goes wrong? Wrong results? Crash? Performance?>\n\n")
	sb.WriteString("  ### Suggested Fix\n")
	sb.WriteString("  ```go\n  // <Code block showing the fix>\n  ```\n")
	sb.WriteString("  (Ensure you still write the reproduction test to disk as Proof of Work, even if you don't explicitly list the command here.)\n")
	sb.WriteString("- If rejected: 'Architectural Intent' explaining why the current behavior is correct.\n")

	if peerTranscript != "" {
		sb.WriteString("\nPeer transcript (Previous Verifier's Work):\n<<<PEER>>>\n")
		sb.WriteString(peerTranscript)
		sb.WriteString("\n<<<END PEER>>>\n")
		sb.WriteString("This is an exchange round. Critique the peer's reasoning. Did they miss the architectural intent? Did they propose an unsafe fix?\n")
	}
	sb.WriteString(fmt.Sprintf("\nRound: %d. Structure your response with headings (Architectural Intent, Code Reasoning, Reproduction Summary, Verdict).\n", round))
	return sb.String()
}

type consensusVerdict struct {
	Agree       bool   `json:"agree"`
	Explanation string `json:"explanation"`
}

func buildConsensusPrompt(issueText string, alpha Transcript, beta Transcript) string {
	var sb strings.Builder
	sb.WriteString("Compare the two codex transcripts below for the same issue.\n\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\nTranscript A:\n")
	sb.WriteString(alpha.Text)
	sb.WriteString("\n\nTranscript B:\n")
	sb.WriteString(beta.Text)
	sb.WriteString("\n\nReply ONLY JSON: {\"agree\":true/false,\"explanation\":\"...\"}. agree=true only if both confirm the same defect and failing test.\n")
	return sb.String()
}

func parseConsensus(raw string) (consensusVerdict, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return consensusVerdict{}, errors.New("empty consensus response")
	}
	jsonBlock := extractJSONBlock(trimmed)
	var verdict consensusVerdict
	if err := json.Unmarshal([]byte(jsonBlock), &verdict); err != nil {
		return consensusVerdict{}, err
	}
	return verdict, nil
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
