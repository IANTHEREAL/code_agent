package prreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func buildReviewPrompt(task, hint string, idx, total int) string {
	var sb strings.Builder
	sb.WriteString("You are review_code, a critical PR reviewer specializing in P0/P1 defects.\n\n")
	sb.WriteString("User Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Run %d of %d. Focus area: %s\n\n", idx, total, hint))
	sb.WriteString("Instructions:\n")
	sb.WriteString("- Inspect the latest workspace state and the PR diff for correctness, regression risks, and missing tests.\n")
	sb.WriteString("- Report ONLY P0 (critical) or P1 (major) defects. Minor/style notes are out of scope.\n")
	sb.WriteString("- For each issue include: title, impacted files, reasoning, and minimal reproduction steps or failing test command.\n")
	sb.WriteString("- If no blocking issues exist, write 'No P0/P1 issues found'.\n")
	sb.WriteString("- Record the findings verbatim into code_review.log at the workspace root.\n")
	sb.WriteString("- Do not modify files; operate strictly read-only for this pass.\n")
	sb.WriteString("- Keep the language concise and actionable; downstream automation will ingest the raw log as-is.\n")
	return sb.String()
}

func buildAggregationPrompt(logs []ReviewerLog) string {
	var sb strings.Builder
	sb.WriteString("Aggregate the following raw review_code logs. Deduplicate overlapping issues and emit ISSUE blocks.\n")
	for i, log := range logs {
		sb.WriteString("\n---\n")
		sb.WriteString(fmt.Sprintf("Review run %d (hint: %s, branch: %s):\n", i+1, log.Hint, log.BranchID))
		sb.WriteString(log.Report)
		sb.WriteString("\n")
	}
	sb.WriteString("\nFormat strictly as:\nISSUE 1: <original issue statement>\nISSUE 2: <...>\n")
	return sb.String()
}

func parseIssueBlocks(raw string) []Issue {
	var issues []Issue
	text := strings.TrimSpace(raw)
	if text == "" {
		return issues
	}
	lines := strings.Split(text, "\n")
	var current Issue
	var body []string

	flush := func() {
		if current.Name == "" {
			return
		}
		statement := strings.TrimSpace(strings.Join(body, "\n"))
		if statement == "" {
			return
		}
		current.Statement = statement
		issues = append(issues, current)
		current = Issue{}
		body = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "ISSUE ") {
			rest := strings.TrimSpace(trimmed[5:])
			num := readLeadingDigits(rest)
			if num != "" {
				flush()
				current = Issue{Name: fmt.Sprintf("ISSUE %s", num)}
				remaining := strings.TrimSpace(rest[len(num):])
				remaining = strings.TrimLeft(remaining, ":- ")
				if remaining != "" {
					body = append(body, remaining)
				} else {
					body = nil
				}
				continue
			}
		}
		if current.Name != "" {
			body = append(body, line)
		}
	}
	flush()
	return issues
}

func readLeadingDigits(s string) string {
	var digits strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		} else {
			break
		}
	}
	return digits.String()
}

func buildCodexPrompt(label, task string, issue Issue, peerTranscript string, round int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are %s, acting as a confirmation reviewer for a GitHub PR.\n\n", label))
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issue.Name)
	sb.WriteString(": ")
	sb.WriteString(issue.Statement)
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

func buildConsensusPrompt(issue Issue, alpha Transcript, beta Transcript) string {
	var sb strings.Builder
	sb.WriteString("Compare the two codex transcripts below for the same issue.\n\n")
	sb.WriteString(issue.Name)
	sb.WriteString(": ")
	sb.WriteString(issue.Statement)
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
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end >= start {
		return raw[start : end+1]
	}
	return raw
}

func buildCommentPrompt(issue Issue, alpha Transcript, beta Transcript) string {
	var sb strings.Builder
	sb.WriteString("Draft a GitHub-ready PR review comment for the confirmed issue.\n\n")
	sb.WriteString("Issue statement:\n")
	sb.WriteString(issue.Name)
	sb.WriteString(": ")
	sb.WriteString(issue.Statement)
	sb.WriteString("\n\nConfirmed evidence:\n")
	sb.WriteString(alpha.Agent)
	sb.WriteString(" says:\n")
	sb.WriteString(alpha.Text)
	sb.WriteString("\n\n")
	sb.WriteString(beta.Agent)
	sb.WriteString(" says:\n")
	sb.WriteString(beta.Text)
	sb.WriteString("\n\nComment requirements:\n")
	sb.WriteString("- Summarize the defect and impact.\n")
	sb.WriteString("- Describe the failing test or reproduction in concrete steps.\n")
	sb.WriteString("- End with a clear ask (e.g., fix, add missing coverage).\n")
	sb.WriteString("- Use Markdown suitable for a GitHub PR review comment.\n")
	return sb.String()
}
