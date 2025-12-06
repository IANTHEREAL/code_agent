package prreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func buildPreparationPrompt(task string) string {
	var sb strings.Builder
	sb.WriteString("You are Codex preparing the workspace for a PR review.\n\n")
	sb.WriteString("PR link / task:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nGoals:\n")
	sb.WriteString("1. Run `gh review` for the PR above to fetch metadata and discover the source branch.\n")
	sb.WriteString("2. Check out that branch locally so the workspace mirrors the PR's code.\n")
	sb.WriteString("3. Stop after confirming the checkout succeeded so subsequent review agents inherit the ready-to-review state.\n")
	sb.WriteString("\nWhen you finish, reply ONLY JSON: {\"branch\":\"<local branch name>\",\"notes\":\"short summary\"}.\n")
	return sb.String()
}

func buildReviewPrompt(task, branch string) string {
	var sb strings.Builder
	sb.WriteString("PR to review:\n")
	sb.WriteString(task)
	trimmedBranch := strings.TrimSpace(branch)
	if trimmedBranch != "" {
		sb.WriteString("\n\nLocal branch already checked out for this PR:\n")
		sb.WriteString(trimmedBranch)
	}
	return sb.String()
}

type prepSummary struct {
	Branch string `json:"branch"`
	Notes  string `json:"notes,omitempty"`
}

func buildAggregationPrompt(logs []ReviewerLog) string {
	var sb strings.Builder
	sb.WriteString("Aggregate the following raw review_code logs. Deduplicate overlapping issues and respond ONLY JSON using this schema:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\"name\":\"ISSUE 1\",\"statement\":\"concise canonical description\",\"source_branches\":[\"branch-id-a\",\"branch-id-b\"]}\n")
	sb.WriteString("]\n")
	sb.WriteString("Populate source_branches with the branch ids for every reviewer that reported the issue. Return [] if no blocking issues remain.\n")
	for i, log := range logs {
		sb.WriteString("\n---\n")
		sb.WriteString(fmt.Sprintf("Review run %d (branch: %s):\n", i+1, log.BranchID))
		sb.WriteString(log.Report)
		sb.WriteString("\n")
	}
	return sb.String()
}

func parseIssueBlocks(raw string) ([]Issue, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	jsonBlock := extractJSONBlock(trimmed)
	var payload []Issue
	if err := json.Unmarshal([]byte(jsonBlock), &payload); err != nil {
		return nil, err
	}
	var issues []Issue
	for _, issue := range payload {
		issue.Name = strings.TrimSpace(issue.Name)
		issue.Statement = strings.TrimSpace(issue.Statement)
		if issue.Name == "" || issue.Statement == "" {
			continue
		}
		var branches []string
		for _, b := range issue.Branches {
			b = strings.TrimSpace(b)
			if b != "" {
				branches = append(branches, b)
			}
		}
		issue.Branches = branches
		issues = append(issues, issue)
	}
	return issues, nil
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

func parsePreparationSummary(raw string) (prepSummary, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return prepSummary{}, errors.New("empty preparation response")
	}
	jsonBlock := extractJSONBlock(trimmed)
	var summary prepSummary
	if err := json.Unmarshal([]byte(jsonBlock), &summary); err != nil {
		return prepSummary{}, err
	}
	summary.Branch = strings.TrimSpace(summary.Branch)
	if summary.Branch == "" {
		return prepSummary{}, errors.New("preparation response missing branch")
	}
	return summary, nil
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
