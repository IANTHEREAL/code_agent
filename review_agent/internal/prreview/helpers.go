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
	sb.WriteString("CRITICAL MINDSET:\n")
	sb.WriteString("- Think deeply and skeptically. Challenge the issue report AND your own assumptions.\n")
	sb.WriteString("- Your goal is to find the TRUTH, not to confirm what you expect.\n")
	sb.WriteString("- If you disagree with prior conclusions, argue against them with logic and evidence.\n\n")
	sb.WriteString("SOURCE OF TRUTH (verify before claiming 'X doesn't exist'):\n")
	sb.WriteString("1. MCP tool definitions > Documentation (check actual tool schemas for valid parameters)\n")
	sb.WriteString("2. Running code > Static docs (code may support features not yet documented)\n")
	sb.WriteString("3. Actual error messages > Assumptions (call it and observe what happens)\n")
	sb.WriteString("- Before claiming something is invalid/unsupported, search for actual usage in the codebase.\n")
	sb.WriteString("- Documentation can be outdated or incomplete; implementation is the ground truth.\n\n")
	sb.WriteString("INSTRUCTIONS:\n")
	sb.WriteString("1. Chesterton's Fence: Before calling it a bug, understand WHY the code exists. Is it protecting against a race condition? Explain the architectural intent.\n")
	sb.WriteString("2. Safety First: If we 'fix' this, what guarantees do we lose? Could it introduce new bugs?\n")
	sb.WriteString("3. Write Tests (Proof of Work): Write a minimal test to disk and run it.\n")
	sb.WriteString("   - EVIDENCE FABRICATION WARNING: Test MUST verify against REAL system behavior.\n")
	sb.WriteString("   - If your mock rejects X, that proves NOTHING about the real system.\n")
	sb.WriteString("   - Valid: real errors, documented specs, actual config. Invalid: self-created mock behavior.\n")
	sb.WriteString("4. Be Skeptical: Assume the code might be right. Only confirm if you can prove it is BROKEN and the fix is SAFE.\n\n")

	// Round-specific guidance
	if round == 1 {
		sb.WriteString("This is Round 1. Independently verify the issue.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("This is Round %d. Build on or challenge prior work.\n\n", round))
	}

	if peerTranscript != "" {
		sb.WriteString("Peer transcript (Previous Verifier's Work):\n<<<PEER>>>\n")
		sb.WriteString(peerTranscript)
		sb.WriteString("\n<<<END PEER>>>\n")
		sb.WriteString("Critique the peer's reasoning. Did they miss the architectural intent? Did they fabricate evidence? Did they propose an unsafe fix?\n\n")
	}

	sb.WriteString("REPORT FORMAT:\n")
	sb.WriteString("If CONFIRMED, use this EXACT structure:\n")
	sb.WriteString("  ## <Short Title>\n\n")
	sb.WriteString("  ### Issue Location\n")
	sb.WriteString("  `path/to/file:line` (can be multiple locations)\n\n")
	sb.WriteString("  ### Problem Summary\n")
	sb.WriteString("  <Concise description of what's wrong>\n\n")
	sb.WriteString("  ### Root Cause Analysis\n")
	sb.WriteString("  <Detailed trace through the code. Use tables, step-by-step lists, or code snippets to show the bug path.>\n\n")
	sb.WriteString("  ### Impact\n")
	sb.WriteString("  <What breaks? Wrong results, crash, performance, scope of affected queries/features.>\n\n")
	sb.WriteString("  ### Suggested Fix\n")
	sb.WriteString("  ```\n  // Code snippet in the project's language\n  ```\n\n")
	sb.WriteString("If REJECTED, explain:\n")
	sb.WriteString("  ### Architectural Intent\n")
	sb.WriteString("  <Why the current code is correct. What did the reporter miss?>\n")
	sb.WriteString("  ### Evidence\n")
	sb.WriteString("  <Test results, documentation, or code analysis proving it's not a bug.>\n")
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
