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
	return sb.String()
}

func buildCodexPrompt(label, task string, issueText string, peerTranscript string, round int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are %s, acting as a confirmation reviewer for a GitHub PR.\n\n", label))
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	sb.WriteString("Goals:\n")
	sb.WriteString("- Inspect the repo and PR diff to understand the suspected defect.\n")
	sb.WriteString("- Describe how to reproduce it locally plus the minimal failing test you would write.\n")
	sb.WriteString("- State whether you CONFIRM the issue as described or believe it is invalid, and justify your stance with evidence.\n")
	sb.WriteString("- Operate read-only; do NOT modify files. Respond with reasoning text only.\n")
	if peerTranscript != "" {
		sb.WriteString("\nPeer transcript:\n<<<PEER>>>\n")
		sb.WriteString(peerTranscript)
		sb.WriteString("\n<<<END PEER>>>\n")
		sb.WriteString("This is an exchange round. Either adopt/improve the peer's failing test, clarify disagreements, or explain why the issue should be withdrawn.\n")
	}
	sb.WriteString(fmt.Sprintf("\nRound: %d. Structure your response with headings (Context, Evidence, Proposed Test, Verdict).\n", round))
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
