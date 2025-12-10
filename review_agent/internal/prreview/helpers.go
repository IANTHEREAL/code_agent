package prreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	verdictConfirmed = "CONFIRMED"
	verdictRejected  = "REJECTED"
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

func buildReviewerPrompt(task, issueText string) string {
	var sb strings.Builder
	sb.WriteString("Simulate a group of senior programmers reviewing this code change.\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(strings.TrimSpace(task))
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(strings.TrimSpace(issueText))
	sb.WriteString("\n\nTheir task:\n")
	sb.WriteString("- Analyze the code logic for correctness.\n")
	sb.WriteString("- Check for edge cases and error handling regressions.\n")
	sb.WriteString("- Understand the architectural intent (Chesterton's Fence) before proposing fixes.\n")
	sb.WriteString("- Identify potential design issues and safety hazards.\n\n")
	sb.WriteString("Evidence standards:\n")
	sb.WriteString("✓ Actual code paths, control-flow diagrams, concrete inputs/outputs.\n")
	sb.WriteString("✓ Git history or specs that prove intent.\n")
	sb.WriteString("✗ Fabricated mocks, \"should\" statements, or gut feelings.\n\n")
	sb.WriteString("Response format:\n")
	sb.WriteString("# VERDICT: CONFIRMED | REJECTED\n")
	sb.WriteString("Provide structured reasoning with Issue Location, Problem Summary, Root Cause, Impact, and Suggested Fix (or Architectural Intent/Evidence for REJECTED).\n")
	return sb.String()
}

func buildTesterPrompt(task, issueText string) string {
	var sb strings.Builder
	sb.WriteString("Simulate a QA engineer who reproduces issues by running code.\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(strings.TrimSpace(task))
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(strings.TrimSpace(issueText))
	sb.WriteString("\n\nTheir task:\n")
	sb.WriteString("- Attempt to reproduce the reported issue end-to-end.\n")
	sb.WriteString("- Write and run a minimal failing test or command sequence.\n")
	sb.WriteString("- Trace actual execution paths and capture stack traces or logs.\n")
	sb.WriteString("- Collect real error messages (no assumptions or mocked output).\n\n")
	sb.WriteString("Output expectations:\n")
	sb.WriteString("# VERDICT: CONFIRMED (with test evidence) | REJECTED (could not reproduce)\n")
	sb.WriteString("Include concrete commands/tests that were run, the observed output, and why it proves or disproves the issue.\n")
	return sb.String()
}

func buildExchangePrompt(role, issueText, peerOpinion string) string {
	var sb strings.Builder
	sb.WriteString("Round 2: Exchange Opinions.\n")
	sb.WriteString("You now have access to your peer's findings. Challenge them and reconcile differences.\n\n")
	sb.WriteString("Role: ")
	sb.WriteString(strings.TrimSpace(role))
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(strings.TrimSpace(issueText))
	sb.WriteString("\n\nPeer Opinion:\n<<<PEER_OPINION>>>\n")
	sb.WriteString(strings.TrimSpace(peerOpinion))
	sb.WriteString("\n<<<END_PEER_OPINION>>>\n\n")
	sb.WriteString("Instructions:\n")
	sb.WriteString("- Identify where your peer's reasoning strengthens or weakens your prior position.\n")
	sb.WriteString("- Resolve contradictions with real evidence (logs, stack traces, code references).\n")
	sb.WriteString("- If evidence is inconclusive, prefer \"存疑不报\" (do not post).\n\n")
	sb.WriteString("Return format:\n")
	sb.WriteString("# VERDICT: CONFIRMED | REJECTED\n")
	sb.WriteString("Provide revised reasoning that explains how the peer feedback changed (or reinforced) your conclusion.\n")
	return sb.String()
}

type consensusVerdict struct {
	Agree       bool   `json:"agree"`
	Explanation string `json:"explanation"`
}

func buildConsensusPrompt(issueText string, alpha Transcript, beta Transcript) string {
	var sb strings.Builder
	sb.WriteString("Compare the Reviewer and Tester transcripts below for the same issue.\n\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\nReviewer Transcript:\n")
	sb.WriteString(alpha.Text)
	sb.WriteString("\n\nTester Transcript:\n")
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

func parseVerdictFromTranscript(text string) (string, error) {
	content := strings.ToUpper(strings.TrimSpace(text))
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# VERDICT:") {
			verdict := strings.TrimSpace(strings.TrimPrefix(trimmed, "# VERDICT:"))
			if isPromptVerdictLine(verdict) {
				continue
			}
			switch {
			case matchesQualifiedVerdict(verdict, verdictConfirmed):
				return verdictConfirmed, nil
			case matchesQualifiedVerdict(verdict, verdictRejected):
				return verdictRejected, nil
			case verdict == "":
				return "", errors.New("empty verdict line")
			default:
				return "", fmt.Errorf("unknown verdict %q", verdict)
			}
		}
	}
	return "", errors.New("verdict marker not found")
}

func matchesQualifiedVerdict(line string, canonical string) bool {
	if !strings.HasPrefix(line, canonical) {
		return false
	}
	if len(line) == len(canonical) {
		return true
	}
	remainder := strings.TrimSpace(line[len(canonical):])
	if remainder == "" {
		return true
	}
	switch remainder[0] {
	case '(':
		if len(remainder) < 2 || !strings.HasSuffix(remainder, ")") {
			return false
		}
		inner := strings.TrimSpace(remainder[1 : len(remainder)-1])
		return inner != ""
	case '-', ':':
		return strings.TrimSpace(remainder[1:]) != ""
	default:
		return false
	}
}

func isPromptVerdictLine(line string) bool {
	return strings.Contains(line, "|") &&
		strings.Contains(line, verdictConfirmed) &&
		strings.Contains(line, verdictRejected)
}

type roundOneOutcome struct {
	NeedExchange bool
	Status       string
}

func evaluateRoundOne(reviewer Transcript, tester Transcript, consensusAgree bool) roundOneOutcome {
	switch {
	case reviewer.Verdict == verdictRejected && tester.Verdict == verdictRejected:
		return roundOneOutcome{NeedExchange: false, Status: commentUnresolved}
	case reviewer.Verdict == verdictConfirmed && tester.Verdict == verdictConfirmed:
		if consensusAgree {
			return roundOneOutcome{NeedExchange: false, Status: commentConfirmed}
		}
		return roundOneOutcome{NeedExchange: true}
	default:
		return roundOneOutcome{NeedExchange: true}
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
