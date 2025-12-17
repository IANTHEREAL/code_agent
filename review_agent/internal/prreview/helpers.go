package prreview

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const universalStudyLine = "Read as much as you can, you have unlimited read quotas and available contexts. When you are not sure about something, you must study the code until you figure out."

func buildIssueFinderPrompt(task string, changeAnalysisPath string) string {
	var sb strings.Builder
	sb.WriteString("Task: ")
	sb.WriteString(task)
	sb.WriteString("\n\n")
	sb.WriteString(universalStudyLine)
	sb.WriteString("\n\n")
	if strings.TrimSpace(changeAnalysisPath) != "" {
		sb.WriteString("Reference (read-only): Change Analysis at: ")
		sb.WriteString(changeAnalysisPath)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Review the code changes against the base branch 'BASE_BRANCH' = main or master (PR-style).\n\n")
	sb.WriteString("  1) Find the merge-base SHA for this comparison:\n")
	sb.WriteString("     - Try: git merge-base HEAD BASE_BRANCH\n")
	sb.WriteString("     - If that fails, try: git merge-base HEAD \"BASE_BRANCH@{upstream}\"\n")
	sb.WriteString("     - If still failing, inspect refs/remotes and pick the correct remote-tracking ref, then re-run merge-base.\n\n")
	sb.WriteString("  2) Once you have MERGE_BASE_SHA, inspect the changes relative to the base branch:\n")
	sb.WriteString("     - Run: git diff MERGE_BASE_SHA\n")
	sb.WriteString("     - Also run: git diff --name-status MERGE_BASE_SHA\n\n")
	sb.WriteString("  3) Provide prioritized, actionable findings based on that diff (correctness, bugs, security, edge cases, API/contracts, tests, UX where relevant). Include file/line references when possible.")
	return sb.String()
}

func buildScoutPrompt(task string, outputPath string) string {
	var sb strings.Builder
	sb.WriteString("Role: SCOUT\n\n")
	sb.WriteString(universalStudyLine)
	sb.WriteString("\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\n")
	sb.WriteString("Requirement: Write a Change Analysis that improves downstream review + testing.\n")
	sb.WriteString("Goal: high-signal summary + impact/risk analysis (NOT a line-by-line commentary).\n")
	sb.WriteString("You MUST base the analysis on an actual diff against base branch (main or master), not assumptions.\n\n")
	sb.WriteString("Get the diff:\n")
	sb.WriteString("  1) Find the merge-base SHA for this comparison:\n")
	sb.WriteString("     - Try: git merge-base HEAD BASE_BRANCH\n")
	sb.WriteString("     - If that fails, try: git merge-base HEAD \"BASE_BRANCH@{upstream}\"\n")
	sb.WriteString("     - If still failing, inspect refs/remotes and pick the correct remote-tracking ref, then re-run merge-base.\n\n")
	sb.WriteString("  2) Once you have MERGE_BASE_SHA, inspect changes relative to the base branch:\n")
	sb.WriteString("     - Run: git diff MERGE_BASE_SHA\n")
	sb.WriteString("     - Also run: git diff --name-status MERGE_BASE_SHA\n\n")
	sb.WriteString("Analysis guidance:\n")
	sb.WriteString("- Focus on behavior, invariants, error semantics, edge cases, concurrency, compatibility.\n")
	sb.WriteString("- If defaults/contracts/config/env/flags changed, treat it as high risk; use `rg` to find likely call sites.\n")
	sb.WriteString("- Include file:line or symbol anchors for key points.\n\n")
	sb.WriteString("Write the analysis to: ")
	sb.WriteString(outputPath)
	sb.WriteString("\n\n")
	sb.WriteString("Output format (concise but complete):\n")
	sb.WriteString("# CHANGE ANALYSIS\n")
	sb.WriteString("## Summary (<= 5 lines)\n")
	sb.WriteString("## Behavioral / Contract Deltas\n")
	sb.WriteString("## High-Risk Areas (ranked)\n")
	sb.WriteString("For each item, include: What changed (anchor), Before -> After, Who/what is impacted, How to verify.\n")
	sb.WriteString("## Impacted Call Sites / Code Paths\n")
	sb.WriteString("## Verification Plan (minimal)\n")
	sb.WriteString("## Appendix: Change Surface\n")
	return sb.String()
}

func buildHasRealIssuePrompt(reportText string) string {
	var sb strings.Builder
	sb.WriteString("You are a strict triage parser for code review reports.\n\n")
	sb.WriteString("Contract:\n")
	sb.WriteString("- If the report explicitly states no P0/P1 issues (or no blockers), treat the PR as clean.\n")
	sb.WriteString("- Otherwise, treat the report as indicating at least one blocking P0/P1 issue.\n\n")
	sb.WriteString("Given the following review report, decide whether it contains a blocking issue.\n")
	sb.WriteString("Reply ONLY with JSON: {\"has_issue\": true} or {\"has_issue\": false}.\n\n")
	sb.WriteString("Review report:\n")
	sb.WriteString(reportText)
	sb.WriteString("\n")
	return sb.String()
}

// buildReviewerPrompt creates the prompt for the Reviewer role (logic analysis).
func buildLogicAnalystPrompt(task string, issueText string, changeAnalysisPath string) string {
	var sb strings.Builder
	sb.WriteString("Verification Role: REVIEWER\n\n")
	sb.WriteString(universalStudyLine)
	sb.WriteString("\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	if strings.TrimSpace(changeAnalysisPath) != "" {
		sb.WriteString("Reference (read-only): Change Analysis at: ")
		sb.WriteString(changeAnalysisPath)
		sb.WriteString("\n\n")
	}
	sb.WriteString("YOUR ROLE: Analyze code logic to determine if this is a valid bug.\n\n")
	sb.WriteString("Simulate a group of senior programmers reviewing this code change.\n\n")
	sb.WriteString("SCOPE RULES (IMPORTANT):\n")
	sb.WriteString("- Your # VERDICT must ONLY judge whether the Issue under review (issueText) claim is real.\n")
	sb.WriteString("- If you notice other problems, include them at the end under: \"## Additions (out of scope)\" and do NOT use them to justify or change your verdict.\n")
	sb.WriteString("- Do NOT claim you ran code/tests; rely on code reading and reasoning only.\n\n")
	sb.WriteString("REQUIRED: After the verdict line, begin with a single-sentence restatement of the issueText claim you are judging.\n\n")
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
	sb.WriteString("\n## Additions (out of scope)\n")
	sb.WriteString("<Optional: other issues you noticed, explicitly out of scope for this verdict>\n")
	return sb.String()
}

// buildTesterPrompt creates the prompt for the Tester role (reproduction).
func buildTesterPrompt(task string, issueText string, changeAnalysisPath string) string {
	var sb strings.Builder
	sb.WriteString("Verification Role: TESTER\n\n")
	sb.WriteString(universalStudyLine)
	sb.WriteString("\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	if strings.TrimSpace(changeAnalysisPath) != "" {
		sb.WriteString("Reference (read-only): Change Analysis at: ")
		sb.WriteString(changeAnalysisPath)
		sb.WriteString("\n\n")
	}
	sb.WriteString("YOUR ROLE: Reproduce the issue by actually running code.\n\n")
	sb.WriteString("Simulate a QA engineer who verifies bugs by running real tests.\n\n")
	sb.WriteString("SCOPE RULES (IMPORTANT):\n")
	sb.WriteString("- Your # VERDICT must ONLY judge whether the Issue under review (issueText) claim exists in reality.\n")
	sb.WriteString("- Your reproduction MUST target that claim directly.\n")
	sb.WriteString("- If you find other failures/issues that are not the issueText claim, include them at the end under: \"## Additions (out of scope)\" and do NOT use them to justify or change your verdict.\n\n")
	sb.WriteString("REQUIRED: After the verdict line, begin with a single-sentence restatement of the issueText claim you are testing.\n\n")
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
	sb.WriteString("If you reference a custom script or test, include the key command or code snippet so others can rerun it; evidence without reproduction detail is not credible.\n")
	sb.WriteString("\n## Additions (out of scope)\n")
	sb.WriteString("<Optional: other issues you noticed, explicitly out of scope for this verdict>\n")
	return sb.String()
}

// buildExchangePrompt creates the prompt for Round 2 (exchange opinions).
func buildExchangePrompt(role string, task string, issueText string, changeAnalysisPath string, selfOpinion string, peerOpinion string) string {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	displayRole := strings.ToUpper(role)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Verification Role: %s (Round 2 - Exchange)\n\n", displayRole))
	sb.WriteString(universalStudyLine)
	sb.WriteString("\n\n")
	sb.WriteString("Task / PR context:\n")
	sb.WriteString(task)
	sb.WriteString("\n\nIssue under review:\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\n")
	if strings.TrimSpace(changeAnalysisPath) != "" {
		sb.WriteString("Reference (read-only): Change Analysis at: ")
		sb.WriteString(changeAnalysisPath)
		sb.WriteString("\n\n")
	}
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
	sb.WriteString("SCOPE RULES (IMPORTANT):\n")
	sb.WriteString("- Your # VERDICT must ONLY judge whether the Issue under review (issueText) claim is real.\n")
	sb.WriteString("- If either opinion mentions other issues, treat them as out of scope: include them under \"## Additions (out of scope)\" and do NOT use them to justify or change your verdict.\n")
	sb.WriteString("- You may change your verdict ONLY based on evidence/reasoning about the issueText claim itself.\n\n")
	sb.WriteString("ROUND 2 REQUIREMENT (KISS):\n")
	sb.WriteString("Immediately after the verdict line, include these two lines:\n")
	sb.WriteString("Claim: <1 sentence restatement of the issueText claim you are judging>\n")
	sb.WriteString("Anchor: <file:line | failing test / repro command | symptom> (use \"unknown\" if not available)\n\n")
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
	sb.WriteString("\n## Additions (out of scope)\n")
	sb.WriteString("<Optional: other issues you noticed, explicitly out of scope for this verdict>\n")
	return sb.String()
}

type verdictDecision struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

var verdictLineRe = regexp.MustCompile(`(?i)^\s*#?\s*verdict\s*:\s*\[?\s*(confirmed|rejected)\s*\]?\s*$`)

func extractTranscriptVerdict(transcript string) (verdictDecision, bool) {
	lines := strings.Split(transcript, "\n")
	limit := 10
	if len(lines) < limit {
		limit = len(lines)
	}
	for i := 0; i < limit; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ">") {
			continue
		}
		matches := verdictLineRe.FindStringSubmatch(line)
		if len(matches) != 2 {
			continue
		}
		return verdictDecision{
			Verdict: strings.ToLower(strings.TrimSpace(matches[1])),
			Reason:  "explicit transcript verdict marker",
		}, true
	}
	return verdictDecision{}, false
}

type alignmentVerdict struct {
	Agree       bool   `json:"agree"`
	Explanation string `json:"explanation"`
}

func buildAlignmentPrompt(issueText string, alpha Transcript, beta Transcript) string {
	var sb strings.Builder
	sb.WriteString("You are aligning two verification transcripts (Reviewer vs Tester) for the SAME issue.\n\n")
	sb.WriteString("Issue under review (issueText):\n")
	sb.WriteString(issueText)
	sb.WriteString("\n\nTranscript A:\n<<<A>>>\n")
	sb.WriteString(alpha.Text)
	sb.WriteString("\n<<<END A>>>\n\nTranscript B:\n<<<B>>>\n")
	sb.WriteString(beta.Text)
	sb.WriteString("\n<<<END B>>>\n\n")
	sb.WriteString("Task:\n")
	sb.WriteString("- Decide whether A and B are confirming/rejecting the SAME issueText claim (same defect).\n")
	sb.WriteString("- Ignore any \"Additions (out of scope)\" sections; they must not affect alignment.\n\n")
	sb.WriteString("Reply ONLY JSON: {\"agree\":true/false,\"explanation\":\"...\"}.\n")
	sb.WriteString("agree=true ONLY if both transcripts are clearly talking about the same underlying defect described by issueText.\n")
	sb.WriteString("If uncertain, return agree=false.\n")
	return sb.String()
}

func parseAlignment(raw string) (alignmentVerdict, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return alignmentVerdict{}, fmt.Errorf("empty alignment response (raw=%q)", truncateForError(raw))
	}
	jsonBlock := extractJSONBlock(trimmed)
	var verdict alignmentVerdict
	if err := json.Unmarshal([]byte(jsonBlock), &verdict); err != nil {
		return alignmentVerdict{}, fmt.Errorf("invalid alignment JSON: %v (json=%q raw=%q)", err, truncateForError(jsonBlock), truncateForError(trimmed))
	}
	return verdict, nil
}

func truncateForError(s string) string {
	const limit = 600
	out := strings.TrimSpace(s)
	if out == "" {
		return ""
	}
	out = strings.ReplaceAll(out, "\n", "\\n")
	out = strings.ReplaceAll(out, "\t", "\\t")
	if len(out) <= limit {
		return out
	}
	return out[:limit] + "...(truncated)"
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
